package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mangabloomapi/cmd/seeder/dto"
	"net/http"

	"github.com/google/uuid"
)

func fetchMangaListWithMetadata(client *http.Client, mangaList *[]Manga, titleList *[]Title,
	authors *[]Author, artists *[]Artist,
	tags *[]Tag, mangaAuthors *[]MangaAuthor,
	mangaArtists *[]MangaArtist, mangaTags *[]MangaTag,
	cover_images *[]string, descriptions *[]Description, page int) error {

	url := fmt.Sprintf("%vmanga?limit=100&offset=%v&order[latestUploadedChapter]=desc", mangadex_api_url, page*100)

	fmt.Println("fetching url : ", url)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch manga list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manga list: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var apiresponse dto.APIResponse
	if err := json.Unmarshal(body, &apiresponse); err != nil {
		log.Fatal(err)
		return err
	}

	for i := 0; i < len(apiresponse.Data); i++ {
		mangaRes := apiresponse.Data[i]
		manga := Manga{
			ID:               uuidParser(mangaRes.ID),
			Title:            keyExistsInMap(&mangaRes.Attributes.Title, "en"),
			Description:      keyExistsInMap(&mangaRes.Attributes.Description, "en"),
			OriginalLanguage: sql.NullString{String: mangaRes.Attributes.OriginalLanguage, Valid: true},
			LastVolume:       sql.NullString{String: mangaRes.Attributes.LastVolume, Valid: true},
			LastChapter:      sql.NullString{String: mangaRes.Attributes.LastChapter, Valid: true},
			Demographic:      sql.NullString{String: mangaRes.Attributes.PublicationDemographic, Valid: true},
			Status:           sql.NullString{String: mangaRes.Attributes.Status, Valid: true},
			Year:             sql.NullInt32{Int32: mangaRes.Attributes.Year, Valid: true},
			ContentRating:    sql.NullString{String: mangaRes.Attributes.ContentRating, Valid: true},
			State:            sql.NullString{String: mangaRes.Attributes.State, Valid: true},
			IsLocked:         sql.NullBool{Bool: mangaRes.Attributes.IsLocked, Valid: true},
			ChapterReset:     sql.NullBool{Bool: mangaRes.Attributes.ChapterNumbersResetOnNewVolume, Valid: true},
			CreatedAt:        sql.NullTime{Time: mangaRes.Attributes.CreatedAt, Valid: true},
			UpdatedAt:        sql.NullTime{Time: mangaRes.Attributes.UpdatedAt, Valid: true},
			Version:          sql.NullInt32{Int32: mangaRes.Attributes.Version, Valid: true},
		}

		populateTitles(mangaRes, titleList)
		populateDescriptions(mangaRes, descriptions)
		populateArtists(mangaRes, artists)
		populateAuthors(mangaRes, authors)
		populateMangaAuthors(mangaRes, mangaAuthors)
		populateMangaArtists(mangaRes, mangaArtists)
		populateTags(mangaRes, tags, mangaTags)
		populateCoverArt(mangaRes, cover_images)
		*mangaList = append(*mangaList, manga)
	}
	return nil
}

func uuidParser(id string) uuid.UUID {
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		log.Fatal("invalid uuid", err)
	}
	return parsedUUID
}

func keyExistsInMap(entityMap *map[string]string, key string) sql.NullString {
	value, exists := (*entityMap)[key]
	if exists {
		return sql.NullString{String: value, Valid: true}
	} else {
		return sql.NullString{String: "", Valid: false}
	}
}

func populateTitles(mangaResponse dto.MangaResponse, titleList *[]Title) {
	for _, altTitle := range mangaResponse.Attributes.AltTitles {
		for language_code, title := range altTitle {
			title := Title{
				MangaID:      uuid.NullUUID{UUID: uuidParser(mangaResponse.ID), Valid: true},
				LanguageCode: sql.NullString{String: language_code, Valid: true},
				Title:        sql.NullString{String: title, Valid: true},
			}
			*titleList = append(*titleList, title)
		}
	}

	for language_code, title := range mangaResponse.Attributes.Title {
		title_temp := Title{
			MangaID:      uuid.NullUUID{UUID: uuidParser(mangaResponse.ID), Valid: true},
			LanguageCode: sql.NullString{String: language_code, Valid: true},
			Title:        sql.NullString{String: title, Valid: true},
		}
		*titleList = append(*titleList, title_temp)
	}
}

func populateDescriptions(mangaResponse dto.MangaResponse, descriptionList *[]Description) {
	for language_code, description := range mangaResponse.Attributes.Description {
		description := Description{
			MangaID:      uuid.NullUUID{UUID: uuidParser(mangaResponse.ID), Valid: true},
			LanguageCode: sql.NullString{String: language_code, Valid: true},
			Description:  sql.NullString{String: description, Valid: true},
		}
		*descriptionList = append(*descriptionList, description)
	}
}

func populateArtists(mangaResponse dto.MangaResponse, artistList *[]Artist) {
	for _, artist := range mangaResponse.Relationships {
		{
			if artist.Type == "artist" {
				artist_temp := Artist{
					ID:   uuidParser(artist.ID),
					Name: artist.ID,
				}
				*artistList = append(*artistList, artist_temp)
			}
		}
	}
}

func populateAuthors(mangaResponse dto.MangaResponse, authorList *[]Author) {
	for _, author := range mangaResponse.Relationships {
		if author.Type == "author" {
			author_temp := Author{
				ID:   uuidParser(author.ID),
				Name: author.ID,
			}
			*authorList = append(*authorList, author_temp)
		}
	}
}

func populateMangaAuthors(mangaResponse dto.MangaResponse, mangaAuthorList *[]MangaAuthor) {
	mangaID := uuidParser(mangaResponse.ID)
	for _, author := range mangaResponse.Relationships {
		if author.Type == "author" {
			authorID := uuidParser(author.ID)
			mangaAuthor := MangaAuthor{
				MangaID:  mangaID,
				AuthorID: authorID,
			}
			*mangaAuthorList = append(*mangaAuthorList, mangaAuthor)
		}
	}
}

func populateMangaArtists(mangaResponse dto.MangaResponse, mangaArtistList *[]MangaArtist) {
	mangaID := uuidParser(mangaResponse.ID)
	for _, artist := range mangaResponse.Relationships {
		if artist.Type == "artist" {
			artistID := uuidParser(artist.ID)
			mangaArtist := MangaArtist{
				MangaID:  mangaID,
				ArtistID: artistID,
			}
			*mangaArtistList = append(*mangaArtistList, mangaArtist)
		}
	}
}

func populateCoverArt(mangaResponse dto.MangaResponse, coverList *[]string) {
	for _, cover := range mangaResponse.Relationships {
		if cover.Type == "cover_art" {
			coverID := uuidParser(cover.ID)
			coverFetchUrl := fmt.Sprintf("https://api.mangadex.org/cover/%v", coverID)
			*coverList = append(*coverList, coverFetchUrl)
		}
	}
}

func populateTags(mangaResponse dto.MangaResponse, tags *[]Tag, mangaTags *[]MangaTag) {
	for _, tag := range mangaResponse.Attributes.Tags {
		tag_temp := Tag{
			ID:          uuidParser(tag.ID),
			Name:        keyExistsInMap(&tag.Attributes.Name, "en"),
			Description: keyExistsInMap(&tag.Attributes.Description, "en"),
			GroupName:   sql.NullString{String: tag.Attributes.Group, Valid: true},
			Version:     sql.NullInt32{Int32: tag.Attributes.Version, Valid: true},
		}
		mangaTag := MangaTag{
			MangaID: uuidParser(mangaResponse.ID),
			TagID:   tag_temp.ID,
		}
		*mangaTags = append(*mangaTags, mangaTag)
		*tags = append(*tags, tag_temp)
	}
}
