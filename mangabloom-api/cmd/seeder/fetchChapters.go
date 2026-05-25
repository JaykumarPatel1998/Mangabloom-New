package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mangabloomapi/cmd/seeder/dto"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func ToNullString(ptr *string) sql.NullString {
	if ptr != nil {
		return sql.NullString{String: *ptr, Valid: true}
	}
	return sql.NullString{Valid: false}
}

func fetchChapters(client *http.Client, mangaId string, chapters *[]Chapter, page int) error {
	url := fmt.Sprintf("%vmanga/%v/feed?limit=100&offset=%v&includeFuturePublishAt=0&includeExternalUrl=0", mangadex_api_url, mangaId, page*100)

	resp, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch chapter list: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResponse dto.ChapterPaginatedResponse

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	for i := 0; i < len(apiResponse.Data); i++ {
		chapterRes := apiResponse.Data[i]

		var mangaId uuid.NullUUID
		for _, rel := range chapterRes.Relationships {
			if rel.Type == "manga" {
				mangaId.UUID = uuidParser(rel.ID)
				mangaId.Valid = true
			}
		}

		chapter := Chapter{
			ID:                 uuidParser(chapterRes.ID),
			MangaID:            mangaId,
			Volume:             ToNullString(chapterRes.Attributes.Volume),
			Chapter:            sql.NullString{String: chapterRes.Attributes.Chapter, Valid: chapterRes.Attributes.Chapter != ""},
			Title:              ToNullString(chapterRes.Attributes.Title),
			TranslatedLanguage: sql.NullString{String: chapterRes.Attributes.TranslatedLanguage, Valid: true},
			ExternalUrl:        ToNullString(chapterRes.Attributes.ExternalURL),
			PublishAt:          sql.NullTime{Time: chapterRes.Attributes.PublishAt, Valid: true},
			ReadableAt:         sql.NullTime{Time: chapterRes.Attributes.ReadableAt, Valid: true},
			CreatedAt:          sql.NullTime{Time: chapterRes.Attributes.CreatedAt, Valid: true},
			UpdatedAt:          sql.NullTime{Time: chapterRes.Attributes.UpdatedAt, Valid: true},
			Pages:              sql.NullInt32{Int32: int32(chapterRes.Attributes.Pages), Valid: chapterRes.Attributes.Pages > 0},
			Version:            sql.NullInt32{Int32: int32(chapterRes.Attributes.Version), Valid: true},
		}

		*chapters = append(*chapters, chapter)
	}
	return nil
}

func PopulateChapters(client *http.Client, mangaId string, chapters *[]Chapter) {
	url := fmt.Sprintf("%vmanga/%v/feed?limit=1&includeFuturePublishAt=0&includeExternalUrl=0", mangadex_api_url, mangaId)

	resp, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("non OK HTTP status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var apiResponse dto.ChapterPaginatedResponse

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Fatal(err)
	}

	total_chapters := apiResponse.Total
	fmt.Printf("Total chapters to fetch : %v\n", total_chapters)

	batchSize := 100
	iterations := (total_chapters + batchSize - 1) / batchSize

	for i := 0; i < iterations; i++ {
		sleep(200 * time.Millisecond)

		err := fetchChapters(client, mangaId, chapters, i)
		if err != nil {
			fmt.Println("Error fetching manga list: ", err)
			return
		}
	}
}
