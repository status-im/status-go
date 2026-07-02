local status_go_root = vim.fn.fnamemodify(debug.getinfo(1, "S").source:sub(2), ":p:h")
local native_root = vim.fs.dirname(status_go_root)
local logos_storage_lib_dir = vim.env.LOGOS_STORAGE_LIB_DIR or native_root .. "/logos-storage-nim/build"
local logos_storage_inc_dir = vim.env.LOGOS_STORAGE_INC_DIR or native_root .. "/logos-storage-nim/library"
local nim_sds_lib_dir = vim.env.NIM_SDS_LIB_DIR or native_root .. "/nim-sds/build"
local nim_sds_inc_dir = vim.env.NIM_SDS_INC_DIR or native_root .. "/nim-sds/library"
local cgo_cflags = vim.env.CGO_CFLAGS and vim.env.CGO_CFLAGS .. " " or ""
local cgo_ldflags = vim.env.CGO_LDFLAGS and vim.env.CGO_LDFLAGS .. " " or ""

local function append_unique_path_list(existing, paths)
	local seen = {}
	local result = {}

	for path in string.gmatch(existing or "", "[^:]+") do
		if not seen[path] then
			table.insert(result, path)
			seen[path] = true
		end
	end

	for _, path in ipairs(paths) do
		if not seen[path] then
			table.insert(result, path)
			seen[path] = true
		end
	end

	return table.concat(result, ":")
end

local native_env = {
	LOGOS_STORAGE_LIB_DIR = logos_storage_lib_dir,
	LOGOS_STORAGE_INC_DIR = logos_storage_inc_dir,
	NIM_SDS_LIB_DIR = nim_sds_lib_dir,
	NIM_SDS_INC_DIR = nim_sds_inc_dir,
	CGO_CFLAGS = cgo_cflags .. "-I" .. logos_storage_inc_dir .. " -I" .. nim_sds_inc_dir,
	CGO_LDFLAGS = cgo_ldflags .. "-L" .. logos_storage_lib_dir .. " -lstorage -Wl,-rpath," .. logos_storage_lib_dir .. " -L" .. nim_sds_lib_dir .. " -lsds",
	LD_LIBRARY_PATH = append_unique_path_list(vim.env.LD_LIBRARY_PATH, { logos_storage_lib_dir, nim_sds_lib_dir }),
}

for key, value in pairs(native_env) do
	vim.env[key] = value
end

vim.lsp.config("gopls", {
	cmd_env = native_env,
	settings = {
		gopls = {
			env = native_env,
			analyses = {
				ST1000 = false,
				ST1003 = false,
			},
			buildFlags = {
				"-tags=gowaku_skip_migrations,gowaku_no_rln,use_logos_storage,use_torrent",
			},
		},
	},
})

local function find_golangci_lint()
	for dir in string.gmatch(vim.env.PATH or "", "[^:]+") do
		local exe = dir .. "/golangci-lint"
		if vim.fn.executable(exe) == 1 and not string.find(exe, "/mason/", 1, true) then
			return exe
		end
	end
end

local function remove_golangci_lint(lint)
	local go_linters = lint.linters_by_ft.go
	if not go_linters then
		return
	end

	local filtered = {}
	for _, linter in ipairs(go_linters) do
		if linter ~= "golangcilint" then
			table.insert(filtered, linter)
		end
	end
	lint.linters_by_ft.go = filtered
end

local function configure_golangci_lint()
	local ok, lint = pcall(require, "lint")
	if not ok or not lint.linters.golangcilint then
		return
	end

	local golangci_lint = find_golangci_lint()
	if not golangci_lint then
		remove_golangci_lint(lint)
		return
	end

	lint.linters.golangcilint.cmd = golangci_lint
	lint.linters.golangcilint.cwd = status_go_root
	lint.linters.golangcilint.args = {
		"--build-tags",
		"gowaku_no_rln lint",
		"run",
		"./...",
		"--output.json.path=stdout",
		"--output.text.path=",
		"--output.tab.path=",
		"--output.html.path=",
		"--output.checkstyle.path=",
		"--output.code-climate.path=",
		"--output.junit-xml.path=",
		"--output.teamcity.path=",
		"--output.sarif.path=",
		"--issues-exit-code=0",
		"--show-stats=false",
		"--path-mode=abs",
	}
end

configure_golangci_lint()

vim.api.nvim_create_autocmd("User", {
	pattern = "VeryLazy",
	callback = configure_golangci_lint,
})
