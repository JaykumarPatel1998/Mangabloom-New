package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mangabloomapi/cmd/seeder/dto"
	"mangabloomapi/internal/db"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var dbConnection *sql.DB

type DBConfig struct {
	DB *db.Queries
}

var mangadex_api_url string

func sleep(duration time.Duration) {
	time.Sleep(duration)
}

func initialize() {
	godotenv.Load(".env")
	db_url_string := os.Getenv("DB_URL")
	mangadex_api_url = os.Getenv("MANGADEX_API_BASE_URL")

	if db_url_string == "" {
		log.Fatal("missing DB_URL in environment variables")
	}

	if mangadex_api_url == "" {
		log.Fatal("missing MANGADEX_API_BASE_URL in environment variables")
	}

	fmt.Println("db url : ", db_url_string)
	fmt.Println("mangadex api url : ", mangadex_api_url)

	db_url, _ := url.Parse(db_url_string)

	var err error
	fmt.Println("connecting to database: ", db_url.String())
	dbConnection, err = sql.Open("postgres", db_url.String())
	if err != nil {
		log.Fatal(err)
	}

	dbConnection.SetMaxOpenConns(20)
	dbConnection.SetConnMaxIdleTime(0)
	dbConnection.SetMaxIdleConns(5)
}

func seedDatabase() {
	initialize()

	migrationTable := &MigrationTable{}
	fmt.Println("migration begins: ", time.Now())

	migrationStart(migrationTable)

	//lets try to query the data and add it to the database

	client := &http.Client{}
	db_cfg := &DBConfig{
		DB: db.New(dbConnection),
	}

	url := fmt.Sprintf("%vmanga?limit=1&order[latestUploadedChapter]=desc", mangadex_api_url)
	resp, err := client.Get(url)
	if err != nil {
		log.Fatal("failed to fetch manga data: ", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("failed to fetch manga data: ", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("failed to read response body: ", err)
	}

	var apiResponse dto.APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Fatal(err)
	}

	// count of total mangas in the response
	totalMangas := apiResponse.Total
	fmt.Printf("total mangas: %d\n", totalMangas)

	batch_size := 100
	iterations := (totalMangas + (batch_size - 1)) / batch_size

	fmt.Println("Total iterations required to get all mangas", iterations)

	for page := 0; page < iterations; page++ {
		var mangaList []Manga
		var titleList []Title
		var tags []Tag
		var authors []Author
		var artists []Artist
		var manga_authors []MangaAuthor
		var manga_artists []MangaArtist
		var cover_images []string
		var descriptions []Description
		var manga_tags []MangaTag

		fmt.Println("fetching manga page number: ", page)
		err := fetchMangaListWithMetadata(client, &mangaList, &titleList, &authors, &artists, &tags, &manga_authors, &manga_artists, &manga_tags, &cover_images, &descriptions, page)
		if err != nil {
			fmt.Println("Error fetching manga list:", err)
			return
		}

		//insert mangas
		for _, manga := range mangaList {
			err := db_cfg.DB.InsertManga(context.Background(), db.InsertMangaParams(manga))
			if err != nil {
				log.Fatal(err)
				return
			}

			var chapter_list []Chapter

			PopulateChapters(client, manga.ID.String(), &chapter_list)
			for _, chapter := range chapter_list {
				err := db_cfg.DB.InsertChapter(context.Background(), db.InsertChapterParams(chapter))
				if err != nil {
					log.Fatal(err)
					return
				}
			}
		}

		//insert titles
		for _, title := range titleList {
			err := db_cfg.DB.InsertTitle(context.Background(), db.InsertTitleParams(title))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		// insert descriptions
		for _, description := range descriptions {
			err := db_cfg.DB.InsertDescription(context.Background(), db.InsertDescriptionParams(description))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		// isnert authors
		for _, author := range authors {
			err := db_cfg.DB.InsertAuthor(context.Background(), db.InsertAuthorParams(author))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		// insert artists
		for _, artist := range artists {
			err := db_cfg.DB.InsertArtist(context.Background(), db.InsertArtistParams(artist))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		// insert manga_authors
		for _, manga_author := range manga_authors {
			err := db_cfg.DB.InsertMangaAuthor(context.Background(), db.InsertMangaAuthorParams(manga_author))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		// insert manga_artists
		for _, manga_artist := range manga_artists {
			err := db_cfg.DB.InsertMangaArtist(context.Background(), db.InsertMangaArtistParams(manga_artist))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		for _, cover_image := range cover_images {
			resp, err := client.Get(cover_image)
			if err != nil {
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}
			sleep(200 * time.Millisecond)

			var apiResponse dto.CoverAPIResponse
			if err := json.Unmarshal(body, &apiResponse); err != nil {
				log.Fatal(err)
				return
			}

			coverRes := apiResponse.Data
			for _, rel := range coverRes.Relationships {
				if rel.Type == "manga" {
					cover := CoverImage{
						ID: uuidParser(coverRes.ID),
						MangaID: uuid.NullUUID{
							UUID:  uuidParser(rel.ID),
							Valid: true,
						},
						FilePath: fmt.Sprintf("https://uploads.mangadex.org/covers/%v/%v.256.jpg", rel.ID, coverRes.Attributes.FileName),
						UploadedAt: sql.NullTime{
							Time:  coverRes.Attributes.CreatedAt,
							Valid: true,
						},
					}

					err := db_cfg.DB.InsertCoverImage(context.Background(), db.InsertCoverImageParams(cover))
					if err != nil {
						log.Fatal(err)
						return
					}
				}
			}
		}

		for _, tag := range tags {
			err := db_cfg.DB.InsertTag(context.Background(), db.InsertTagParams(tag))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		for _, manga_tag := range manga_tags {
			err := db_cfg.DB.InsertMangaTag(context.Background(), db.InsertMangaTagParams(manga_tag))
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		//cleanup
		sleep(200 * time.Millisecond) // Delay after each page fetch
	}

}

func main() {
	seedDatabase()
}
