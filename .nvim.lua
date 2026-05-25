vim.lsp.config("gopls", {
	settings = {
		gopls = {
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
