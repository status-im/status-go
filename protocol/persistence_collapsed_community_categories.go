package protocol

func (p *sqlitePersistence) UpsertCollapsedCommunityCategory(category CollapsedCommunityCategory) error {
	var err error
	if category.Collapsed {
		_, err = p.db.Exec("INSERT INTO collapsed_community_categories(community_id, category_id) VALUES(?,?)", category.CommunityID, category.CategoryID)
	} else {
		_, err = p.db.Exec("DELETE FROM collapsed_community_categories WHERE community_id = ? AND category_id = ?", category.CommunityID, category.CategoryID)
	}
	return err
}

func (p *sqlitePersistence) CollapsedCommunityCategories() ([]CollapsedCommunityCategory, error) {
	var categories []CollapsedCommunityCategory

	rows, err := p.db.Query(`
  SELECT
    community_id,
    category_id
  FROM
    collapsed_community_categories
  `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c CollapsedCommunityCategory
		err = rows.Scan(
			&c.CommunityID,
			&c.CategoryID,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}
