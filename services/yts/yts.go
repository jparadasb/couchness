package yts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/utils"
)

const (
	// ServiceType yts service type
	ServiceType = "yts"
)

var trackers = []string{
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.openbittorrent.com:80",
	"udp://tracker.coppersurfer.tk:6969",
	"udp://glotorrents.pw:6969/announce",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://torrent.gresille.org:80/announce",
	"udp://p4p.arenabg.com:1337",
	"udp://tracker.leechers-paradise.org:6969",
}

// Service YTS movie service (imdb lookups only)
type Service struct {
	ID      string   `json:"id"`
	BaseURL string   `json:"base_url"`
	Mirrors []string `json:"mirrors"`
	Client  *http.Client
}

type response struct {
	Status string `json:"status"`
	Data   struct {
		MovieCount int     `json:"movie_count"`
		Movies     []movie `json:"movies"`
	} `json:"data"`
}

type movie struct {
	Title     string    `json:"title"`
	TitleLong string    `json:"title_long"`
	Year      int       `json:"year"`
	ImdbCode  string    `json:"imdb_code"`
	Torrents  []torrent `json:"torrents"`
}

type torrent struct {
	Hash      string `json:"hash"`
	Quality   string `json:"quality"`
	Type      string `json:"type"`
	Seeds     int    `json:"seeds"`
	Peers     int    `json:"peers"`
	SizeBytes int64  `json:"size_bytes"`
}

// GetID get service ID
func (s Service) GetID() string {
	return s.ID
}

// GetURL get service base URL
func (s Service) GetURL() string {
	return s.BaseURL
}

// ShowURL get movie information URL
func (s Service) ShowURL(showID string, page, limit int) string {
	return fmt.Sprintf("%s?query_term=%s&page=%d&limit=%d", s.BaseURL, url.QueryEscape(showID), page, limit)
}

// GetShowData get movie torrents from YTS. YTS only indexes movies.
func (s Service) GetShowData(show *models.Show, page, limit int, typeOf string) (*models.Show, error) {
	if show.ExternalID == "" {
		return nil, errors.New("Show " + show.ID + " doesn't have a external ID")
	}
	if typeOf != "movies" {
		return show, nil
	}

	result, err := s.query(show.ExternalID, page, limit)
	if err != nil {
		return show, err
	}

	for _, m := range result.Data.Movies {
		if !strings.EqualFold(m.ImdbCode, show.ExternalID) {
			continue
		}
		for _, t := range m.Torrents {
			name := fmt.Sprintf("%s.%d.%s.%s.YTS", strings.ReplaceAll(m.Title, " ", "."), m.Year, t.Quality, strings.ToUpper(t.Type))
			torrentInfo, err := utils.ParseTorrent(name)
			if err != nil {
				continue
			}
			torrentInfo.Title = fmt.Sprintf("%s [%s %s] YTS", m.TitleLong, t.Quality, t.Type)
			torrentInfo.MagnetURL = Magnet(t.Hash, name)
			torrentInfo.Downloaded = false
			torrentInfo.Seeds = t.Seeds
			torrentInfo.Size = t.SizeBytes
			torrentInfo.Resolution = t.Quality
			show.Episodes = append(show.Episodes, torrentInfo)
		}
		show.TorrentCount += len(m.Torrents)
	}

	return show, nil
}

// query tries every mirror until one answers with a valid response
func (s Service) query(imdbID string, page, limit int) (*response, error) {
	mirrors := append([]string{s.BaseURL}, s.Mirrors...)
	var lastErr error
	for _, base := range mirrors {
		result, err := s.queryMirror(base, imdbID, page, limit)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (s Service) queryMirror(base, imdbID string, page, limit int) (*response, error) {
	url := fmt.Sprintf("%s?query_term=%s&page=%d&limit=%d", base, url.QueryEscape(imdbID), page, limit)
	resp, err := s.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("yts returned HTTP %d", resp.StatusCode)
	}
	result := &response{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, err
	}
	if result.Status != "ok" {
		return nil, errors.New("yts returned status " + result.Status)
	}
	return result, nil
}

// Magnet builds a magnet link from a torrent hash
func Magnet(hash, name string) string {
	magnet := "magnet:?xt=urn:btih:" + hash + "&dn=" + url.QueryEscape(name)
	for _, tracker := range trackers {
		magnet += "&tr=" + url.QueryEscape(tracker)
	}
	return magnet
}

// New create new yts service
func New() Service {
	return Service{
		ID:      ServiceType,
		BaseURL: "https://yts.mx/api/v2/list_movies.json",
		Mirrors: []string{
			"https://movies-api.accel.li/api/v2/list_movies.json",
			"https://yts.gg/api/v2/list_movies.json",
		},
		Client: &http.Client{Timeout: 20 * time.Second},
	}
}
