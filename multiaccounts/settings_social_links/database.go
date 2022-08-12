package sociallinkssettings

import (
	"context"
	"database/sql"
	"errors"

	"github.com/status-im/status-go/protocol/identity"
)

type SocialLinksSettings struct {
	db *sql.DB
}

func NewSocialLinksSettings(db *sql.DB) *SocialLinksSettings {
	return &SocialLinksSettings{
		db: db,
	}
}

func (s *SocialLinksSettings) GetSocialLinks() (identity.SocialLinks, error) {
	rows, err := s.db.Query(`SELECT link_text, link_url FROM social_links_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := identity.SocialLinks{}

	for rows.Next() {
		link := identity.SocialLink{}
		var url sql.NullString

		err := rows.Scan(
			&link.Text, &url,
		)
		if err != nil {
			return nil, err
		}

		if url.Valid {
			link.URL = url.String
		}
		links = append(links, link)
	}

	return links, nil
}

// Be careful, it removes every row except static links (__twitter, etc.) before insertion
func (s *SocialLinksSettings) SetSocialLinks(links *identity.SocialLinks) (err error) {
	if links == nil {
		return errors.New("can't set social links, nil object provided")
	}

	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	// remove everything except static links
	_, err = tx.Exec(`DELETE from social_links_settings WHERE link_text != ? AND link_text != ? AND link_text != ? AND link_text != ? AND link_text != ? AND link_text != ?`,
		identity.TwitterID, identity.PersonalSiteID, identity.GithubID, identity.YoutubeID, identity.DiscordID, identity.TelegramID)

	stmt, err := tx.Prepare("INSERT INTO social_links_settings (link_text, link_url) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, link := range *links {
		_, err = stmt.Exec(
			link.Text,
			link.URL,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
