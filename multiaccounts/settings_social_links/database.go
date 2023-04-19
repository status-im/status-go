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

func (s *SocialLinksSettings) GetSocialLink(text string) (*identity.SocialLink, error) {
	rows, err := s.db.Query(`SELECT link_text, link_url, clock FROM social_links_settings WHERE link_text = ?`, text)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		link := identity.SocialLink{}
		var url sql.NullString

		err = rows.Scan(
			&link.Text, &url, &link.Clock,
		)
		if err != nil {
			return nil, err
		}

		if url.Valid {
			link.URL = url.String
		}
		return &link, nil
	}

	return nil, nil
}

func (s *SocialLinksSettings) GetSocialLinks() (identity.SocialLinks, error) {
	rows, err := s.db.Query(`SELECT link_text, link_url, clock FROM social_links_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := identity.SocialLinks{}

	for rows.Next() {
		link := identity.SocialLink{}
		var url sql.NullString

		err := rows.Scan(
			&link.Text, &url, &link.Clock,
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

// Be careful, it removes every row before insertion
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

	// remove everything
	_, err = tx.Exec(`DELETE from social_links_settings`)

	stmt, err := tx.Prepare("INSERT INTO social_links_settings (link_text, link_url, clock) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, link := range *links {
		_, err = stmt.Exec(
			link.Text,
			link.URL,
			link.Clock,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SocialLinksSettings) UpdateSocialLinkFromSync(link *identity.SocialLink) error {
	if link == nil {
		return errors.New("can't update social link, nil object provided")
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

	stmt, err := tx.Prepare("UPDATE social_links_settings SET link_text = ?, link_url = ?, clock = ? WHERE link_text = ? AND clock < ?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(link.Text, link.URL, link.Clock, link.Text, link.Clock)
	if err != nil {
		return err
	}
	return stmt.Close()
}
