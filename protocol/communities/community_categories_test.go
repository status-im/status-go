package communities

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func (s *CommunitySuite) TestCreateCategory() {
	newCategoryID := "new-category-id"
	newCategoryName := "new-category-name"

	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = nil
	org.config.ID = nil

	_, err := org.CreateCategory(newCategoryID, newCategoryName, []string{})
	s.Require().Equal(ErrNotAuthorized, err)

	org.config.PrivateKey = s.identity
	org.config.ID = &s.identity.PublicKey

	changes, err := org.CreateCategory(newCategoryID, newCategoryName, []string{})

	description := org.config.CommunityDescription

	s.Require().NoError(err)
	s.Require().NotNil(description.Categories)
	s.Require().NotNil(description.Categories[newCategoryID])
	s.Require().Equal(newCategoryName, description.Categories[newCategoryID].Name)
	s.Require().Equal(newCategoryID, description.Categories[newCategoryID].CategoryId)
	s.Require().Equal(int32(len(description.Categories)-1), description.Categories[newCategoryID].Position)
	s.Require().NotNil(changes)
	s.Require().NotNil(changes.CategoriesAdded[newCategoryID])
	s.Require().Equal(description.Categories[newCategoryID], changes.CategoriesAdded[newCategoryID])
	s.Require().Nil(changes.CategoriesModified[newCategoryID])

	_, err = org.CreateCategory(newCategoryID, newCategoryName, []string{})
	s.Require().Equal(ErrCategoryAlreadyExists, err)

	newCategoryID2 := "new-category-id2"
	newCategoryName2 := "new-category-name2"

	changes, err = org.CreateCategory(newCategoryID2, newCategoryName2, []string{})
	s.Require().NoError(err)
	s.Require().Equal(int32(len(description.Categories)-1), description.Categories[newCategoryID2].Position)
	s.Require().NotNil(changes.CategoriesAdded[newCategoryID2])
	s.Require().Nil(changes.CategoriesModified[newCategoryID2])

	newCategoryID3 := "new-category-id3"
	newCategoryName3 := "new-category-name3"
	_, err = org.CreateCategory(newCategoryID3, newCategoryName3, []string{"some-chat-id"})
	s.Require().Equal(ErrChatNotFound, err)

	newChatID := "new-chat-id"
	identity := &protobuf.ChatIdentity{
		DisplayName: "new-chat-display-name",
		Description: "new-chat-description",
	}
	permissions := &protobuf.CommunityPermissions{
		Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	_, err = org.CreateChat(newChatID, &protobuf.CommunityChat{
		Identity:    identity,
		Permissions: permissions,
	})
	s.Require().NoError(err)

	changes, err = org.CreateCategory(newCategoryID3, newCategoryName3, []string{newChatID})
	s.Require().NoError(err)
	s.Require().NotNil(changes.ChatsModified[newChatID])
	s.Require().Equal(newCategoryID3, changes.ChatsModified[newChatID].CategoryModified)

	newCategoryID4 := "new-category-id4"
	newCategoryName4 := "new-category-name4"

	_, err = org.CreateCategory(newCategoryID4, newCategoryName4, []string{newChatID})
	s.Require().Equal(ErrChatAlreadyAssigned, err)
}

func (s *CommunitySuite) TestEditCategory() {
	newCategoryID := "new-category-id"
	newCategoryName := "new-category-name"
	editedCategoryName := "edited-category-name"

	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = s.identity
	_, err := org.CreateCategory(newCategoryID, newCategoryName, []string{testChatID1})
	s.Require().NoError(err)
	org.config.PrivateKey = nil
	org.config.ID = nil

	_, err = org.EditCategory(newCategoryID, editedCategoryName, []string{testChatID1})
	s.Require().Equal(ErrNotAuthorized, err)

	org.config.PrivateKey = s.identity
	org.config.ID = &s.identity.PublicKey

	_, err = org.EditCategory("some-random-category", editedCategoryName, []string{testChatID1})
	s.Require().Equal(ErrCategoryNotFound, err)

	changes, err := org.EditCategory(newCategoryID, editedCategoryName, []string{testChatID1})

	description := org.config.CommunityDescription

	s.Require().NoError(err)
	s.Require().Equal(editedCategoryName, description.Categories[newCategoryID].Name)
	s.Require().NotNil(changes)
	s.Require().NotNil(changes.CategoriesModified[newCategoryID])
	s.Require().Equal(description.Categories[newCategoryID], changes.CategoriesModified[newCategoryID])
	s.Require().Nil(changes.CategoriesAdded[newCategoryID])

	_, err = org.EditCategory(newCategoryID, editedCategoryName, []string{"some-random-chat-id"})
	s.Require().Equal(ErrChatNotFound, err)

	_, err = org.EditCategory(testCategoryID1, testCategoryName1, []string{testChatID1})
	s.Require().Equal(ErrChatAlreadyAssigned, err)

	// Edit by removing the chats

	identity1 := &protobuf.ChatIdentity{
		DisplayName: "new-chat-display-name",
		Description: "new-chat-description",
	}
	permissions := &protobuf.CommunityPermissions{
		Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	testChatID2 := "test-chat-id-2"
	testChatID3 := "test-chat-id-3"

	_, err = org.CreateChat(testChatID2, &protobuf.CommunityChat{
		Identity:    identity1,
		Permissions: permissions,
	})
	s.Require().NoError(err)
	identity2 := &protobuf.ChatIdentity{
		DisplayName: "identity-2",
		Description: "new-chat-description",
	}

	_, err = org.CreateChat(testChatID3, &protobuf.CommunityChat{
		Identity:    identity2,
		Permissions: permissions,
	})
	s.Require().NoError(err)

	_, err = org.EditCategory(newCategoryID, editedCategoryName, []string{testChatID1, testChatID2, testChatID3})
	s.Require().NoError(err)

	s.Require().Equal(newCategoryID, description.Chats[testChatID1].CategoryId)
	s.Require().Equal(newCategoryID, description.Chats[testChatID2].CategoryId)
	s.Require().Equal(newCategoryID, description.Chats[testChatID3].CategoryId)

	s.Require().Equal(int32(0), description.Chats[testChatID1].Position)
	s.Require().Equal(int32(1), description.Chats[testChatID2].Position)
	s.Require().Equal(int32(2), description.Chats[testChatID3].Position)

	_, err = org.EditCategory(newCategoryID, editedCategoryName, []string{testChatID1, testChatID3})
	s.Require().NoError(err)
	s.Require().Equal("", description.Chats[testChatID2].CategoryId)
	s.Require().Equal(int32(0), description.Chats[testChatID1].Position)
	s.Require().Equal(int32(1), description.Chats[testChatID3].Position)

	_, err = org.EditCategory(newCategoryID, editedCategoryName, []string{testChatID3})
	s.Require().NoError(err)
	s.Require().Equal("", description.Chats[testChatID1].CategoryId)
	s.Require().Equal(int32(0), description.Chats[testChatID3].Position)
}

func (s *CommunitySuite) TestDeleteCategory() {
	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = s.identity
	permissions := &protobuf.CommunityPermissions{
		Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	testChatID2 := "test-chat-id-2"
	testChatID3 := "test-chat-id-3"
	newCategoryID := "new-category-id"
	newCategoryName := "new-category-name"

	identity1 := &protobuf.ChatIdentity{
		DisplayName: "display-name-2",
		Description: "new-chat-description",
	}

	_, err := org.CreateChat(testChatID2, &protobuf.CommunityChat{
		Identity:    identity1,
		Permissions: permissions,
	})
	s.Require().NoError(err)

	identity2 := &protobuf.ChatIdentity{
		DisplayName: "display-name-3",
		Description: "new-chat-description",
	}

	_, err = org.CreateChat(testChatID3, &protobuf.CommunityChat{
		Identity:    identity2,
		Permissions: permissions,
	})
	s.Require().NoError(err)

	_, err = org.CreateCategory(newCategoryID, newCategoryName, []string{})
	s.Require().NoError(err)

	description := org.config.CommunityDescription

	_, err = org.EditCategory(newCategoryID, newCategoryName, []string{testChatID2, testChatID1})
	s.Require().NoError(err)
	s.Require().Equal(newCategoryID, description.Chats[testChatID1].CategoryId)
	s.Require().Equal(newCategoryID, description.Chats[testChatID2].CategoryId)

	s.Require().Equal(int32(0), description.Chats[testChatID3].Position)
	s.Require().Equal(int32(0), description.Chats[testChatID2].Position)
	s.Require().Equal(int32(1), description.Chats[testChatID1].Position)

	org.config.PrivateKey = nil
	org.config.ID = nil
	_, err = org.DeleteCategory(testCategoryID1)
	s.Require().Equal(ErrNotAuthorized, err)

	org.config.PrivateKey = s.identity
	org.config.ID = &s.identity.PublicKey
	_, err = org.DeleteCategory("some-category-id")
	s.Require().Equal(ErrCategoryNotFound, err)

	changes, err := org.DeleteCategory(newCategoryID)
	s.Require().NoError(err)
	s.Require().NotNil(changes)

	s.Require().Equal("", description.Chats[testChatID1].CategoryId)
	s.Require().Equal("", description.Chats[testChatID2].CategoryId)
	s.Require().Equal("", description.Chats[testChatID3].CategoryId)
}

func (s *CommunitySuite) TestDeleteChatOrder() {
	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = s.identity
	permissions := &protobuf.CommunityPermissions{
		Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	testChatID2 := "test-chat-id-2"
	testChatID3 := "test-chat-id-3"
	newCategoryID := "new-category-id"
	newCategoryName := "new-category-name"

	identity1 := &protobuf.ChatIdentity{
		DisplayName: "identity-1",
		Description: "new-chat-description",
	}

	_, err := org.CreateChat(testChatID2, &protobuf.CommunityChat{
		Identity:    identity1,
		Permissions: permissions,
	})
	s.Require().NoError(err)
	identity2 := &protobuf.ChatIdentity{
		DisplayName: "identity-2",
		Description: "new-chat-description",
	}

	_, err = org.CreateChat(testChatID3, &protobuf.CommunityChat{
		Identity:    identity2,
		Permissions: permissions,
	})
	s.Require().NoError(err)

	_, err = org.CreateCategory(newCategoryID, newCategoryName, []string{testChatID1, testChatID2, testChatID3})
	s.Require().NoError(err)

	changes, err := org.DeleteChat(testChatID2)
	s.Require().NoError(err)
	s.Require().Equal(int32(0), org.Chats()[testChatID1].Position)
	s.Require().Equal(int32(1), org.Chats()[testChatID3].Position)
	s.Require().Len(changes.ChatsRemoved, 1)

	changes, err = org.DeleteChat(testChatID1)
	s.Require().NoError(err)
	s.Require().Equal(int32(0), org.Chats()[testChatID3].Position)
	s.Require().Len(changes.ChatsRemoved, 1)

	_, err = org.DeleteChat(testChatID3)
	s.Require().NoError(err)
}
