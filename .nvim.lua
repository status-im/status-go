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
