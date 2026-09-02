package common

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/utils"
	"github.com/highercomve/couchness/utils/humanize"
)

const sepator = "="

// GetMovies list every movie stored in the database
func GetMovies() (movies []*models.Movie, err error) {
	movies, err = storage.GetAllMovies()
	if err != nil {
		return nil, err
	}

	sort.SliceStable(movies, func(i, j int) bool {
		return strings.ToLower(movies[i].Show.Title) < strings.ToLower(movies[j].Show.Title)
	})

	return movies, nil
}

// SearchMovieTorrents look up every available torrent of a movie on its services, best seeded first
func SearchMovieTorrents(movie *models.Movie) (models.Episodes, error) {
	if movie.ExternalID == "" {
		return nil, errors.New("movie " + movie.Show.Title + " doesn't have an IMDb ID")
	}
	if movie.Configuration == nil {
		movie.Configuration = &models.ShowConf{}
	}
	if len(movie.Configuration.Services) == 0 && movie.Configuration.Service == "" {
		movie.Configuration.Services = DefaultMovieServices
	}

	torrents, err := getMovieVersionFromServices(movie, getShowServices(&movie.Show), 1, 50, "movies")
	if err != nil {
		return nil, err
	}

	torrents = utils.GetEpisodeVersion(
		torrents,
		-1,
		-1,
		strings.ToLower(movie.Configuration.Codec),
		strings.ToLower(movie.Configuration.Resolution),
		strings.ToLower(movie.Configuration.Quality),
	)

	sort.SliceStable(torrents, func(i, j int) bool {
		return torrents[i].Seeds > torrents[j].Seeds
	})

	if len(torrents) == 0 {
		return nil, fmt.Errorf("no torrents found for movie: %s", movie.Summary())
	}

	return torrents, nil
}

// DownloadMovie queue the selected torrent on transmission and store the movie
func DownloadMovie(movie *models.Movie, torrent *models.TorrentInfo) error {
	if torrent == nil || torrent.MagnetURL == "" {
		return errors.New("a torrent with a magnet link is required")
	}
	if movie.Directory == "" {
		return errors.New("movie directory is required")
	}

	movie.TorrentInfo = *torrent
	movie.TorrentInfo.Location = movie.Directory
	movie.TorrentInfo.Downloaded = true

	if _, err := DownloadTorrent(torrent.MagnetURL, movie.Directory); err != nil {
		return err
	}

	_, err := storage.NewMovieStorage(movie).Save()
	return err
}

// RemoveMovie removes a movie record only.
// Media files and Transmission torrents remain untouched.
func RemoveMovie(movieID string) (*models.Movie, error) {
	movie := &models.Movie{}
	if err := storage.Db.Driver.Read(storage.Db.Collections.Movies, movieID, movie); err != nil {
		return nil, err
	}
	if err := storage.DeleteMovie(movieID); err != nil {
		return nil, err
	}
	return movie, nil
}

func getTorrents(movie *models.Movie, channel chan<- models.Episodes, errChannel chan<- error) {
	torrents, err := SearchMovieTorrents(movie)
	if err != nil {
		errChannel <- err
		return
	}

	channel <- torrents
}

// AddMovie search a movie on the services, ask the user to pick a version and download it
func AddMovie(movie *models.Movie) (*models.Movie, error) {
	fmt.Printf(
		"\n\r\n\r %s Select the movie version of %s: %s \n\r\n\r",
		strings.Repeat(sepator, 5),
		movie.ExternalID,
		strings.Repeat(sepator, 5),
	)

	tChannel := make(chan models.Episodes)
	errChannel := make(chan error)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	go getTorrents(movie, tChannel, errChannel)

	fmt.Print("\033[s")
	var torrents models.Episodes

	i := 0
mainLoop:
	for {
		select {
		case torrents = <-tChannel:
			fmt.Print("\033[u\033[K")
			fmt.Println("")
			break mainLoop
		case err := <-errChannel:
			fmt.Print("\033[u\033[K")
			fmt.Println("")
			return nil, err
		case <-ticker.C:
			i++
			fmt.Print("\033[u\033[K")
			fmt.Printf("Searching torrents %s", strings.Repeat(".", (i/3)+1))
		}
	}

	fmt.Println("")

	table := utils.PrintTable([]string{"#", "Name", "Size", "Seeds"}, nil)
	for i, torrent := range torrents {
		table.Append([]string{
			strconv.Itoa(i + 1),
			torrent.Title,
			humanize.Bytes(uint64(torrent.Size)),
			strconv.Itoa(torrent.Seeds),
		})
	}
	table.Render()

	var input int
	for {
		fmt.Printf("Select a version between 1 and %d: ", len(torrents))
		if _, err := fmt.Scan(&input); err != nil {
			return nil, fmt.Errorf("invalid selection: %s", err.Error())
		}
		if input >= 1 && input <= len(torrents) {
			break
		}
		fmt.Printf("Please select between 1 and %d \n\r\n\r", len(torrents))
	}

	if err := DownloadMovie(movie, torrents[input-1]); err != nil {
		return movie, err
	}

	return movie, nil
}
