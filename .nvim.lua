vim.lsp.config("gopls", {
  settings = {
    gopls = {
      buildFlags = {
        "-tags=gowaku_skip_migrations,gowaku_no_rln,use_logos_storage,use_torrent",
      },
    },
  },
})
