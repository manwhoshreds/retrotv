package server

import "net/http"

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Settings
	mux.HandleFunc("GET /api/settings", s.handleSettings)
	mux.HandleFunc("PUT /api/settings", s.handleUpdateSettings)

	// Startup
	mux.HandleFunc("GET /api/client/startup", s.handleClientStartup)
	mux.HandleFunc("GET /api/dashboard/storage", s.handleDashboardStorage)

	// Scanning
	mux.HandleFunc("GET /api/scan/status", s.handleScanStatus)
	mux.HandleFunc("POST /api/scan", s.handleMovieScan)
	mux.HandleFunc("POST /api/scan/tv", s.handleTVShowScan)
	mux.HandleFunc("POST /api/scan/movies", s.handleMovieScan)
	mux.HandleFunc("POST /api/scan/tvshows", s.handleTVShowScan)

	// Movies
	mux.HandleFunc("DELETE /api/movies/{id}", s.handleDeleteMovie)
	mux.HandleFunc("GET /api/movies", s.handleGetMovies)
	mux.HandleFunc("GET /api/movies/{id}/metadata-matches", s.handleMovieMetadataMatches)
	mux.HandleFunc("POST /api/movies/{id}/refresh-metadata", s.handleRefreshMovieMetadata)

	// TV Shows
	mux.HandleFunc("DELETE /api/tvshows/{id}", s.handleDeleteTVShow)
	mux.HandleFunc("GET /api/tvshows", s.handleGetTVShows)
	mux.HandleFunc("GET /api/tvshows/{id}/metadata-matches", s.handleTVShowMetadataMatches)
	mux.HandleFunc("POST /api/tvshows/{id}/refresh-metadata", s.handleRefreshTVShowMetadata)

	// Metadata
	mux.HandleFunc("GET /api/metadata/facets", s.handleMetadataFacets)

	// Collections
	mux.HandleFunc("GET /api/collections", s.handleGetCollections)
	mux.HandleFunc("POST /api/collections", s.handleCreateCollection)
	mux.HandleFunc("PUT /api/collections/{id}", s.handleUpdateCollection)
	mux.HandleFunc("DELETE /api/collections/{id}", s.handleDeleteCollection)
	mux.HandleFunc("POST /api/collections/preview", s.handlePreviewCollection)
	mux.HandleFunc("POST /api/collection-artwork", s.handleUploadCollectionArtwork)

	// Channel Assets
	mux.HandleFunc("GET /api/assets", s.handleGetAssets)
	mux.HandleFunc("POST /api/assets/scan", s.handleAssetScan)

	// Channels
	mux.HandleFunc("DELETE /api/channels/{id}", s.handleDeleteChannel)
	mux.HandleFunc("GET /api/channels", s.handleGetChannels)
	mux.HandleFunc("GET /api/channels/{id}", s.handleGetChannel)
	mux.HandleFunc("GET /api/channels/{id}/schedule", s.handleChannelSchedule)
	mux.HandleFunc("GET /api/channels/{id}/tune", s.handleTuneChannel)
	mux.HandleFunc("POST /api/channel-logos", s.handleUploadChannelLogo)
	mux.HandleFunc("GET /api/channels/export", s.handleExportChannels)
	mux.HandleFunc("GET /api/channels/import/status", s.handleChannelImportStatus)
	mux.HandleFunc("POST /api/channels/import", s.handleImportChannels)
	mux.HandleFunc("POST /api/channels", s.handleCreateChannel)
	mux.HandleFunc("PUT /api/channels/reorder", s.handleReorderChannels)
	mux.HandleFunc("PUT /api/channels/{id}", s.handleUpdateChannel)
	mux.HandleFunc("PUT /api/channels/{id}/programming", s.handleUpdateChannelProgramming)
	mux.HandleFunc("PUT /api/channels/{id}/airing", s.handleSetChannelAiring)

	// System
	mux.HandleFunc("GET /api/system/disks", s.handleDisksInfo)

	// Guide
	mux.HandleFunc("GET /guide.xml", s.handleGuideXMLTV)
	mux.HandleFunc("GET /channels.m3u", s.handleChannelsM3U)
	mux.HandleFunc("GET /channels/{id}/xmltv", s.handleChannelXMLTV)
	mux.HandleFunc("GET /channels/{id}/m3u", s.handleChannelM3U)

	// Media Streaming (legacy direct playback)
	mux.HandleFunc("GET /media/movies/{id}", s.handleMovieStream)
	mux.HandleFunc("GET /media/tvshows/{id}", s.handleTVShowStream)
	mux.HandleFunc("GET /media/assets/{id}", s.handleAssetStream)

	// Live Channel Streaming (HLS)
	mux.HandleFunc("GET /streams/channels/{id}/index.m3u8", s.handleHLSPlaylist)
	mux.HandleFunc("GET /streams/channels/{id}/{segment}", s.handleHLSSegment)

	// Static files
	mux.Handle("/artwork/", http.StripPrefix("/artwork/", http.FileServer(http.Dir(s.artworkDir))))
	mux.Handle("/logos/", http.StripPrefix("/logos/", http.FileServer(http.Dir(s.legacyLogoDir))))
	mux.Handle("/", noCache(http.FileServer(http.Dir(s.staticDir))))

	return mux
}
