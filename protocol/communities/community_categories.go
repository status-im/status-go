package communities

import (
	"sort"

	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) CreateCategory(categoryID string, categoryName string, chatIDs []string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if o.config.CommunityDescription.Categories == nil {
		o.config.CommunityDescription.Categories = make(map[string]*protobuf.CommunityCategory)
	}
	if _, ok := o.config.CommunityDescription.Categories[categoryID]; ok {
		return nil, ErrCategoryAlreadyExists
	}

	for _, cid := range chatIDs {
		c, exists := o.config.CommunityDescription.Chats[cid]
		if !exists {
			return nil, ErrChatNotFound
		}

		if exists && c.CategoryId != categoryID && c.CategoryId != "" {
			return nil, ErrChatAlreadyAssigned
		}
	}

	changes := o.emptyCommunityChanges()

	o.config.CommunityDescription.Categories[categoryID] = &protobuf.CommunityCategory{
		CategoryId: categoryID,
		Name:       categoryName,
		Position:   int32(len(o.config.CommunityDescription.Categories)),
	}

	for i, cid := range chatIDs {
		o.config.CommunityDescription.Chats[cid].CategoryId = categoryID
		o.config.CommunityDescription.Chats[cid].Position = int32(i)
	}

	o.SortCategoryChats(changes, "")

	o.increaseClock()

	changes.CategoriesAdded[categoryID] = o.config.CommunityDescription.Categories[categoryID]
	for i, cid := range chatIDs {
		changes.ChatsModified[cid] = &CommunityChatChanges{
			MembersAdded:     make(map[string]*protobuf.CommunityMember),
			MembersRemoved:   make(map[string]*protobuf.CommunityMember),
			CategoryModified: categoryID,
			PositionModified: i,
		}
	}

	return changes, nil
}

func (o *Community) EditCategory(categoryID string, categoryName string, chatIDs []string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if o.config.CommunityDescription.Categories == nil {
		o.config.CommunityDescription.Categories = make(map[string]*protobuf.CommunityCategory)
	}
	if _, ok := o.config.CommunityDescription.Categories[categoryID]; !ok {
		return nil, ErrCategoryNotFound
	}

	for _, cid := range chatIDs {
		c, exists := o.config.CommunityDescription.Chats[cid]
		if !exists {
			return nil, ErrChatNotFound
		}

		if exists && c.CategoryId != categoryID && c.CategoryId != "" {
			return nil, ErrChatAlreadyAssigned
		}
	}

	changes := o.emptyCommunityChanges()

	emptyCatLen := o.getCategoryChatCount("")

	// remove any chat that might have been assigned before and now it's not part of the category
	var chatsToRemove []string
	for k, chat := range o.config.CommunityDescription.Chats {
		if chat.CategoryId == categoryID {
			found := false
			for _, c := range chatIDs {
				if k == c {
					found = true
				}
			}
			if !found {
				chat.CategoryId = ""
				chatsToRemove = append(chatsToRemove, k)
			}
		}
	}

	o.config.CommunityDescription.Categories[categoryID].Name = categoryName

	for i, cid := range chatIDs {
		o.config.CommunityDescription.Chats[cid].CategoryId = categoryID
		o.config.CommunityDescription.Chats[cid].Position = int32(i)
	}

	for i, cid := range chatsToRemove {
		o.config.CommunityDescription.Chats[cid].Position = int32(emptyCatLen + i)
		changes.ChatsModified[cid] = &CommunityChatChanges{
			MembersAdded:     make(map[string]*protobuf.CommunityMember),
			MembersRemoved:   make(map[string]*protobuf.CommunityMember),
			CategoryModified: "",
			PositionModified: int(o.config.CommunityDescription.Chats[cid].Position),
		}
	}

	o.SortCategoryChats(changes, "")

	o.increaseClock()

	changes.CategoriesModified[categoryID] = o.config.CommunityDescription.Categories[categoryID]
	for i, cid := range chatIDs {
		changes.ChatsModified[cid] = &CommunityChatChanges{
			MembersAdded:     make(map[string]*protobuf.CommunityMember),
			MembersRemoved:   make(map[string]*protobuf.CommunityMember),
			CategoryModified: categoryID,
			PositionModified: i,
		}
	}

	return changes, nil
}

func (o *Community) ReorderCategories(categoryID string, newPosition int) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if newPosition > 0 && newPosition >= len(o.config.CommunityDescription.Categories) {
		newPosition = len(o.config.CommunityDescription.Categories) - 1
	} else if newPosition < 0 {
		newPosition = 0
	}

	o.config.CommunityDescription.Categories[categoryID].Position = int32(newPosition)

	s := make(sortSlice, 0, len(o.config.CommunityDescription.Categories))
	for catID, category := range o.config.CommunityDescription.Categories {

		position := category.Position
		if category.CategoryId != categoryID && position >= int32(newPosition) {
			position = position + 1
		}

		s = append(s, sorterHelperIdx{
			pos:   position,
			catID: catID,
		})
	}

	changes := o.emptyCommunityChanges()

	o.setModifiedCategories(changes, s)

	o.increaseClock()

	return changes, nil
}

func (o *Community) setModifiedCategories(changes *CommunityChanges, s sortSlice) {
	sort.Sort(s)
	for i, catSortHelper := range s {
		if o.config.CommunityDescription.Categories[catSortHelper.catID].Position != int32(i) {
			o.config.CommunityDescription.Categories[catSortHelper.catID].Position = int32(i)
			changes.CategoriesModified[catSortHelper.catID] = o.config.CommunityDescription.Categories[catSortHelper.catID]
		}
	}
}

func (o *Community) ReorderChat(categoryID string, chatID string, newPosition int) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if _, exists := o.config.CommunityDescription.Categories[categoryID]; !exists {
		return nil, ErrCategoryNotFound
	}

	var chat *protobuf.CommunityChat
	var exists bool
	if chat, exists = o.config.CommunityDescription.Chats[chatID]; !exists {
		return nil, ErrChatNotFound
	}

	oldCategoryID := chat.CategoryId
	chat.CategoryId = categoryID

	changes := o.emptyCommunityChanges()

	o.SortCategoryChats(changes, oldCategoryID)
	o.insertAndSort(changes, categoryID, chat, newPosition)

	o.increaseClock()

	return changes, nil
}

func (o *Community) SortCategoryChats(changes *CommunityChanges, categoryID string) {
	var catChats []string
	for k, c := range o.config.CommunityDescription.Chats {
		if c.CategoryId == categoryID {
			catChats = append(catChats, k)
		}
	}

	sortedChats := make(sortSlice, 0, len(catChats))
	for _, k := range catChats {
		sortedChats = append(sortedChats, sorterHelperIdx{
			pos:    o.config.CommunityDescription.Chats[k].Position,
			chatID: k,
		})
	}

	sort.Sort(sortedChats)

	for i, chatSortHelper := range sortedChats {
		if o.config.CommunityDescription.Chats[chatSortHelper.chatID].Position != int32(i) {
			o.config.CommunityDescription.Chats[chatSortHelper.chatID].Position = int32(i)
			if changes.ChatsModified[chatSortHelper.chatID] != nil {
				changes.ChatsModified[chatSortHelper.chatID].PositionModified = i
			} else {
				changes.ChatsModified[chatSortHelper.chatID] = &CommunityChatChanges{
					PositionModified: i,
					MembersAdded:     make(map[string]*protobuf.CommunityMember),
					MembersRemoved:   make(map[string]*protobuf.CommunityMember),
				}
			}
		}
	}
}

func (o *Community) insertAndSort(changes *CommunityChanges, categoryID string, chat *protobuf.CommunityChat, newPosition int) {
	var catChats []string
	for k, c := range o.config.CommunityDescription.Chats {
		if c.CategoryId == categoryID {
			catChats = append(catChats, k)
		}
	}

	if newPosition > 0 && newPosition >= len(catChats) {
		newPosition = len(catChats) - 1
	} else if newPosition < 0 {
		newPosition = 0
	}

	sortedChats := make(sortSlice, 0, len(catChats))
	for _, k := range catChats {
		position := chat.Position
		if o.config.CommunityDescription.Chats[k] != chat && position >= int32(newPosition) {
			position = position + 1
		}

		sortedChats = append(sortedChats, sorterHelperIdx{
			pos:    position,
			chatID: k,
		})
	}

	sort.Sort(sortedChats)

	for i, chatSortHelper := range sortedChats {
		if o.config.CommunityDescription.Chats[chatSortHelper.chatID].Position != int32(i) {
			o.config.CommunityDescription.Chats[chatSortHelper.chatID].Position = int32(i)
			if changes.ChatsModified[chatSortHelper.chatID] != nil {
				changes.ChatsModified[chatSortHelper.chatID].PositionModified = i
			} else {
				changes.ChatsModified[chatSortHelper.chatID] = &CommunityChatChanges{
					MembersAdded:     make(map[string]*protobuf.CommunityMember),
					MembersRemoved:   make(map[string]*protobuf.CommunityMember),
					PositionModified: i,
				}
			}
		}
	}
}

func (o *Community) getCategoryChatCount(categoryID string) int {
	result := 0
	for _, chat := range o.config.CommunityDescription.Chats {
		if chat.CategoryId == categoryID {
			result = result + 1
		}
	}
	return result
}

func (o *Community) DeleteCategory(categoryID string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if _, exists := o.config.CommunityDescription.Categories[categoryID]; !exists {
		return nil, ErrCategoryNotFound
	}

	changes := o.emptyCommunityChanges()

	emptyCategoryChatCount := o.getCategoryChatCount("")
	i := 0
	for _, chat := range o.config.CommunityDescription.Chats {
		if chat.CategoryId == categoryID {
			i++
			chat.CategoryId = ""
			chat.Position = int32(emptyCategoryChatCount + i)
		}
	}

	o.SortCategoryChats(changes, "")

	delete(o.config.CommunityDescription.Categories, categoryID)

	changes.CategoriesRemoved = append(changes.CategoriesRemoved, categoryID)

	// Reorder
	s := make(sortSlice, 0, len(o.config.CommunityDescription.Categories))
	for _, cat := range o.config.CommunityDescription.Categories {
		s = append(s, sorterHelperIdx{
			pos:   cat.Position,
			catID: cat.CategoryId,
		})
	}

	o.setModifiedCategories(changes, s)

	o.increaseClock()

	return changes, nil
}
