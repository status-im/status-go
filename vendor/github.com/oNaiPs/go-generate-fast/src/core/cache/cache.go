package cache

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/oNaiPs/go-generate-fast/src/core/config"
	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"github.com/oNaiPs/go-generate-fast/src/utils/copy"
	"github.com/oNaiPs/go-generate-fast/src/utils/fs"
	"github.com/oNaiPs/go-generate-fast/src/utils/hash"
	"github.com/oNaiPs/go-generate-fast/src/utils/str"
	"go.uber.org/zap"
)

type VerifyResult struct {
	PluginMatch *plugins.Plugin
	CacheHit    bool
	CacheHitDir string
	CanSave     bool
	IoFiles     plugins.InputOutputFiles
}

func Verify(opts plugins.GenerateOpts) (VerifyResult, error) {
	zap.S().Debugf("%s: verifying cache for \"%s\"", opts.Path, opts.Command())

	verifyResult := VerifyResult{}
	var ioFiles *plugins.InputOutputFiles

	plugin := plugins.MatchPlugin(opts)
	if plugin != nil {
		verifyResult.PluginMatch = &plugin

		zap.S().Debugf("Using plugin \"%s\"", plugin.Name())

		ioFiles = plugin.ComputeInputOutputFiles(opts)
		if ioFiles == nil {
			zap.S().Debugf("No input output files, skipping cache.")
			return verifyResult, nil
		}
	} else {
		zap.S().Debugf("No plugin was found to handle command.")
		ioFiles = &plugins.InputOutputFiles{}

		if len(opts.ExtraInputPatterns) == 0 || len(opts.ExtraOutputPatterns) == 0 {
			return verifyResult, nil
		}
	}

	for _, globPattern := range opts.ExtraInputPatterns {
		matches, err := doublestar.FilepathGlob(globPattern)
		if err != nil {
			zap.S().Error("cannot get extra input files: ", err)
			continue
		}
		ioFiles.InputFiles = append(ioFiles.InputFiles, matches...)
	}

	ioFiles.OutputPatterns = append(ioFiles.OutputPatterns, opts.ExtraOutputPatterns...)

	str.RemoveDuplicatesAndSort(&ioFiles.InputFiles)
	str.RemoveDuplicatesAndSort(&ioFiles.OutputFiles)

	_ = str.ConvertToRelativePaths(&ioFiles.InputFiles, opts.Dir())
	_ = str.ConvertToRelativePaths(&ioFiles.OutputFiles, opts.Dir())

	zap.S().Debugf("Got %d input files: %s", len(ioFiles.InputFiles), strings.Join(ioFiles.InputFiles, ", "))
	zap.S().Debugf("Got %d output files: %s", len(ioFiles.OutputFiles), strings.Join(ioFiles.OutputFiles, ", "))
	zap.S().Debugf("Got %d output globs: %s", len(ioFiles.OutputPatterns), strings.Join(ioFiles.OutputPatterns, ", "))

	cacheHitDir, err := calculateCacheDirectoryFromInputData(opts, *ioFiles)
	if err != nil {
		zap.S().Debugf("Cannot get cache hit dir: %s", err)
		return verifyResult, err
	}

	verifyResult.IoFiles = *ioFiles
	verifyResult.CacheHitDir = cacheHitDir
	zap.S().Debugf("Cache hit dir: %s", cacheHitDir)

	fileInfo, err := os.Stat(cacheHitDir)
	if os.IsNotExist(err) {
		zap.S().Debugf("Cache hit dir not found: %s", cacheHitDir)
	} else if os.IsPermission(err) {
		zap.S().Debugf("Cache hit dir permission denied: %s", cacheHitDir)
	} else if err != nil {
		return VerifyResult{}, fmt.Errorf("cannot get cache dir info: %w", err)
	}

	verifyResult.CacheHit = fileInfo != nil && fileInfo.IsDir()
	verifyResult.CanSave = true

	return verifyResult, nil
}

func Save(result VerifyResult) error {
	outputFiles := result.IoFiles.OutputFiles
	for _, globPattern := range result.IoFiles.OutputPatterns {
		matches, err := doublestar.FilepathGlob(globPattern, doublestar.WithFilesOnly())
		if err != nil {
			zap.S().Error("cannot extra output files: ", err)
			continue
		}
		outputFiles = append(outputFiles, matches...)
	}

	err := os.MkdirAll(result.CacheHitDir, 0700)
	if err != nil {
		return fmt.Errorf("cannot create cache dir: %w", err)
	}

	cacheConfig := CacheConfig{}

	//use an intermediary file since we don't know the file hash until we finish copying it
	tmpFile := path.Join(result.CacheHitDir, "file.swp")

	for _, file := range outputFiles {
		if err != nil {
			return fmt.Errorf("cannot create temp dir: %w", err)
		}

		hash, err := copy.CopyHashFile(file, tmpFile)
		if err != nil {
			return fmt.Errorf("cannot copy file to cache: %w", err)
		}

		err = os.Rename(tmpFile, path.Join(result.CacheHitDir, hash))
		if err != nil {
			return fmt.Errorf("rename file to be cached: %w", err)
		}

		fileStat, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("cannot stat cached file: %w", err)
		}

		cacheConfig.OutputFiles = append(cacheConfig.OutputFiles, CacheConfigOutputFileInfo{
			Hash:    hash,
			Path:    file,
			ModTime: fileStat.ModTime(),
		})
	}

	err = SaveConfig(cacheConfig, result.CacheHitDir)
	if err != nil {
		return fmt.Errorf("cannot write cache config: %w", err)
	}

	zap.S().Debug("Saved cache on ", result.CacheHitDir)

	return nil
}

func Restore(result VerifyResult) error {
	zap.S().Debugf("Restoring cache")

	cacheConfig, err := LoadConfig(result.CacheHitDir)
	if err != nil {
		return fmt.Errorf("cannot read cache config: %w", err)
	}

	// confirm that the expected output files match the ones in the saved cache config
	// we can only do this when there are no globs defined
	// TODO: check if the non-matching output files match the provided glob
	if len(result.IoFiles.OutputPatterns) == 0 &&
		!areOutputsMatching(cacheConfig.OutputFiles, result.IoFiles.OutputFiles) {
		return errors.New("expected output files differ")
	}

	for _, dstFile := range cacheConfig.OutputFiles {
		srcFile := path.Join(result.CacheHitDir, dstFile.Hash)

		// skip if modification time is the same
		dstFileStat, err := os.Stat(dstFile.Path)
		if err == nil && dstFileStat.ModTime() == dstFile.ModTime {
			zap.S().Debug("Skipping copy of file with same modtime: ", dstFile.Path)
			continue
		}

		err = os.MkdirAll(path.Dir(dstFile.Path), 0755)
		if err != nil {
			return fmt.Errorf("cannot create destination directory: %w", err)
		}

		hash, err := copy.CopyHashFile(srcFile, dstFile.Path)
		if err != nil {
			return fmt.Errorf("cannot copy file from cache: %w", err)
		}
		zap.S().Debug("Copied file from cache: ", dstFile.Path)

		err = os.Chtimes(dstFile.Path, dstFile.ModTime, dstFile.ModTime)
		if err != nil {
			return fmt.Errorf("cannot restore times for destination file: %w", err)
		}

		if hash != dstFile.Hash {
			return errors.New("file hash is different, corruption")
		}
	}

	return nil
}

func areOutputsMatching(outputFiles []CacheConfigOutputFileInfo, resultFiles []string) bool {
	// Create a map for faster lookup.
	resultFileMap := make(map[string]bool)
	for _, file := range resultFiles {
		resultFileMap[file] = true
	}

	// Check if each value in outputFiles is present in the resultFileMap.
	for _, value := range outputFiles {
		if !resultFileMap[value.Path] {
			return false
		}
	}

	return true
}

func calculateCacheDirectoryFromInputData(opts plugins.GenerateOpts, ioFiles plugins.InputOutputFiles) (string, error) {
	contentToHash :=
		opts.Dir() +
			strings.Join(opts.Words, "\n") +
			strings.Join(ioFiles.InputFiles, "\n") +
			strings.Join(ioFiles.OutputFiles, "\n") +
			strings.Join(ioFiles.OutputPatterns, "\n") +
			strings.Join(ioFiles.Extra, "\n")

	for _, file := range ioFiles.InputFiles {
		hash, err := hash.HashFile(file)
		if err != nil {
			return "", fmt.Errorf("cannot hash file '%s': %w", file, err)
		}
		contentToHash += hash
	}

	if opts.GoPackage == "" {
		execInfo, err := getExecutableDetails(opts.ExecutableName)
		if err != nil {
			return "", fmt.Errorf("cannot get path for executable '%s': %s", opts.ExecutableName, err)
		}
		contentToHash += execInfo
	} else {
		// we can only hash specific versions/hashes
		if opts.GoPackageVersion != "" && opts.GoPackageVersion != "latest" {
			hash, err := hash.HashString(opts.GoPackage + "/" + opts.GoPackageVersion)
			if err != nil {
				return "", fmt.Errorf("cannot hash string: %w", err)
			}
			contentToHash += hash
		}
	}

	finalHash, err := hash.HashString(contentToHash)
	if err != nil {
		return "", fmt.Errorf("cannot get final hash: %s", err)
	}

	cacheHitDir := path.Join(
		config.Get().CacheDir,
		finalHash[0:1],
		finalHash[1:3],
		finalHash[3:])

	return cacheHitDir, nil
}

func getExecutableDetails(ExecutablePath string) (string, error) {
	ExecutablePath, err := fs.FindExecutablePath(ExecutablePath)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(ExecutablePath)
	if err != nil {
		return "", err
	}

	execInfo := fmt.Sprint(
		ExecutablePath,
		fmt.Sprintf("%019d", info.Size()),
		info.ModTime().Format(time.RFC3339))

	zap.S().Debugf("Exec info %s", execInfo)

	return execInfo, nil
}
