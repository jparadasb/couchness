package tpb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/utils"
)

const (
	// ServiceType pirate bay service type
	ServiceType = "tpb"
)

var (
	movieCategories = map[string]bool{"201": true, "202": true, "207": true, "209": true, "211": true}
	showCategories  = map[string]bool{"205": true, "208": true, "212": true}
	trackers        = []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://open.stealth.si:80/announce",
		"udp://tracker.torrent.eu.org:451/announce",
		"udp://tracker.bittor.pw:1337/announce",
		"udp://public.popcorn-tracker.org:6969/announce",
		"udp://tracker.dler.org:6969/announce",
		"udp://exodus.desync.com:6969",
		"udp://opentracker.i2p.rocks:6969/announce",
	}
)

// Service The Pirate Bay JSON API (apibay)
type Service struct {
	ID      string `json:"id"`
	BaseURL string `json:"base_url"`
	Client  *http.Client
}

type result struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Seeders  string `json:"seeders"`
	Size     string `json:"size"`
	Category string `json:"category"`
	Imdb     string `json:"imdb"`
}

// GetID get service ID
func (s Service) GetID() string {
	return s.ID
}

// GetURL get service base URL
func (s Service) GetURL() string {
	return s.BaseURL
}

// ShowURL get search URL for an imdb ID
func (s Service) ShowURL(showID string, page, limit int) string {
	return fmt.Sprintf("%s?q=%s&cat=0", s.BaseURL, url.QueryEscape(showID))
}

// GetShowData search torrents by imdb ID
func (s Service) GetShowData(show *models.Show, page, limit int, typeOf string) (*models.Show, error) {
	if show.ExternalID == "" {
		return nil, errors.New("Show " + show.ID + " doesn't have a external ID")
	}

	resp, err := s.Client.Get(s.ShowURL(show.ExternalID, page, limit))
	if err != nil {
		return show, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return show, fmt.Errorf("tpb returned HTTP %d", resp.StatusCode)
	}

	results := []result{}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return show, err
	}

	categories := showCategories
	if typeOf == "movies" {
		categories = movieCategories
	}

	count := 0
	for _, r := range results {
		if r.ID == "0" || r.InfoHash == "" {
			continue
		}
		if !categories[r.Category] {
			continue
		}
		if r.Imdb != "" && !strings.EqualFold(r.Imdb, show.ExternalID) {
			continue
		}
		torrentInfo, err := utils.ParseTorrent(r.Name)
		if err != nil {
			continue
		}
		torrentInfo.Title = r.Name
		torrentInfo.MagnetURL = Magnet(r.InfoHash, r.Name)
		torrentInfo.Downloaded = false
		torrentInfo.Location = show.Directory + r.Name
		torrentInfo.Seeds, _ = strconv.Atoi(r.Seeders)
		torrentInfo.Size, _ = strconv.ParseInt(r.Size, 10, 64)
		show.Episodes = append(show.Episodes, torrentInfo)
		count++
		if limit > 0 && count >= limit {
			break
		}
	}
	show.TorrentCount += count

	return show, nil
}

// Magnet builds a magnet link from an info hash
func Magnet(hash, name string) string {
	magnet := "magnet:?xt=urn:btih:" + strings.ToLower(hash) + "&dn=" + url.QueryEscape(name)
	for _, tracker := range trackers {
		magnet += "&tr=" + url.QueryEscape(tracker)
	}
	return magnet
}

// New create new pirate bay service
func New() Service {
	return Service{
		ID:      ServiceType,
		BaseURL: "https://apibay.org/q.php",
		Client:  &http.Client{Timeout: 20 * time.Second},
	}
}
