package server

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"retrotv/internal/models"
)

func (s *Server) handleGuideXMLTV(w http.ResponseWriter, r *http.Request) {
	channels, err := s.db.Channels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	channels = airingChannels(channels)
	baseURL := requestBaseURL(r)
	start, end := xmltvWindow(r, time.Now().UTC())
	channelIDs := make([]int64, 0, len(channels))
	for _, channel := range channels {
		channelIDs = append(channelIDs, channel.ID)
	}
	programsByChannel, err := s.db.GuideScheduledProgramsWindow(r.Context(), channelIDs, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := xmltvTV{
		Channels:   []xmltvChannel{},
		Programmes: []xmltvProgramme{},
	}
	for _, channel := range channels {
		channelResponse := xmltvResponse(channel, programsByChannel[channel.ID], baseURL)
		response.Channels = append(response.Channels, channelResponse.Channels...)
		response.Programmes = append(response.Programmes, channelResponse.Programmes...)
	}

	payload, err := xml.MarshalIndent(response, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(payload)
}

func (s *Server) handleChannelsM3U(w http.ResponseWriter, r *http.Request) {
	channels, err := s.db.Channels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	channels = airingChannels(channels)

	baseURL := requestBaseURL(r)
	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "#EXTM3U")
	for _, channel := range channels {
		_, _ = fmt.Fprintf(w, "#EXTINF:-1 tvg-id=\"retrotv-%d\" tvg-name=\"%s\" tvg-logo=\"%s\",%s\n%s/channels/%d/m3u\n",
			channel.ID,
			escapeM3U(channel.Name),
			escapeM3U(publicURL(baseURL, channel.LogoPath)),
			escapeM3U(channel.Name),
			baseURL,
			channel.ID,
		)
	}
}

func (s *Server) handleChannelXMLTV(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	channel, err := s.db.Channel(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	var programs []models.ScheduledProgram
	if _, ok := r.URL.Query()["hours"]; ok {
		start, end := xmltvWindow(r, time.Now().UTC())
		programsByChannel, guideErr := s.db.GuideScheduledProgramsWindow(r.Context(), []int64{id}, start, end)
		programs, err = programsByChannel[id], guideErr
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		programs, err = s.db.ScheduledPrograms(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	payload, err := xml.MarshalIndent(xmltvResponse(channel, programs, requestBaseURL(r)), "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(payload)
}

func (s *Server) handleChannelM3U(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	channel, err := s.db.Channel(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	programs, err := s.db.ScheduledPrograms(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	baseURL := requestBaseURL(r)

	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "#EXTM3U")
	for _, program := range currentAndFuturePrograms(programs, time.Now().UTC()) {
		duration := int(program.EndTime.Sub(program.StartTime).Seconds())
		_, _ = fmt.Fprintf(w, "#EXTINF:%d tvg-id=\"retrotv-%d\" tvg-name=\"%s\" tvg-logo=\"%s\",%s\n%s\n",
			duration,
			channel.ID,
			escapeM3U(channel.Name),
			escapeM3U(publicURL(baseURL, channel.LogoPath)),
			escapeM3U(programTitle(program)),
			s.renderPlayback(baseURL, program, 0).URL,
		)
	}
}
