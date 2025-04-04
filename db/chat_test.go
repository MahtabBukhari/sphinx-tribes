package db

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetChatsForWorkspace(t *testing.T) {
	InitTestDB()

	TestDB.db.Exec("DELETE FROM chats")

	chats := []Chat{
		{ID: "chat1", WorkspaceID: "workspace123", Status: ActiveStatus},
		{ID: "chat2", WorkspaceID: "workspace123", Status: ActiveStatus},
		{ID: "chat3", WorkspaceID: "workspace123", Status: ActiveStatus},
		{ID: "chat4", WorkspaceID: "workspace123", Status: ActiveStatus},
		{ID: "chat5", WorkspaceID: "workspace123", Status: ActiveStatus},
	}

	for _, chat := range chats {
		result := TestDB.db.Create(&chat)
		assert.NoError(t, result.Error)
	}

	tests := []struct {
		name        string
		workspaceID string
		limit       int
		offset      int
		expectedLen int
		totalCount  int64
		expectError bool
	}{
		{
			name:        "Basic Functionality - No Pagination",
			workspaceID: "workspace123",
			limit:       -1,
			offset:      0,
			expectedLen: 5,
			totalCount:  5,
			expectError: false,
		},
		{
			name:        "With Pagination - First Page",
			workspaceID: "workspace123",
			limit:       2,
			offset:      0,
			expectedLen: 2,
			totalCount:  5,
			expectError: false,
		},
		{
			name:        "With Pagination - Second Page",
			workspaceID: "workspace123",
			limit:       2,
			offset:      2,
			expectedLen: 2,
			totalCount:  5,
			expectError: false,
		},
		{
			name:        "With Pagination - Last Page",
			workspaceID: "workspace123",
			limit:       2,
			offset:      4,
			expectedLen: 1,
			totalCount:  5,
			expectError: false,
		},
		{
			name:        "No Chats for Workspace",
			workspaceID: "emptyWorkspace",
			limit:       -1,
			offset:      0,
			expectedLen: 0,
			totalCount:  0,
			expectError: false,
		},
		{
			name:        "Empty Workspace ID",
			workspaceID: "",
			limit:       -1,
			offset:      0,
			expectedLen: 0,
			totalCount:  0,
			expectError: true,
		},
		{
			name:        "Invalid Offset",
			workspaceID: "workspace123",
			limit:       2,
			offset:      10,
			expectedLen: 0,
			totalCount:  5,
			expectError: false,
		},
		{
			name:        "SQL Injection Attempt",
			workspaceID: "workspace123'; DROP TABLE chats; --",
			limit:       -1,
			offset:      0,
			expectedLen: 0,
			totalCount:  0,
			expectError: false,
		},
		{
			name:        "Unicode Characters in Workspace ID",
			workspaceID: "工作区123",
			limit:       -1,
			offset:      0,
			expectedLen: 0,
			totalCount:  0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chats, total, err := TestDB.GetChatsForWorkspace(tt.workspaceID, "", tt.limit, tt.offset)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLen, len(chats), "Expected %d chats, got %d", tt.expectedLen, len(chats))
				assert.Equal(t, tt.totalCount, total, "Expected total count of %d, got %d", tt.totalCount, total)

				if len(chats) > 0 {
					assert.Equal(t, tt.workspaceID, chats[0].WorkspaceID)
				}
			}
		})
	}

	TestDB.db.Exec("DELETE FROM chats")
}

func TestGetChatMessagesForChatID(t *testing.T) {
	InitTestDB()
	currentTime := time.Now()

	tests := []struct {
		name        string
		setup       func() string
		expected    []ChatMessage
		expectError bool
	}{
		{
			name: "Successfully geting messages for chat",
			setup: func() string {
				chatID := "chat123"
				messages := []ChatMessage{
					{
						ID:        "msg1",
						ChatID:    chatID,
						Message:   "Hello",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
					{
						ID:        "msg2",
						ChatID:    chatID,
						Message:   "Hi there",
						Role:      AssistantRole,
						Timestamp: currentTime.Add(time.Minute),
						Status:    SentStatus,
						Source:    AgentSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "msg1",
					ChatID:  "chat123",
					Message: "Hello",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
				{
					ID:      "msg2",
					ChatID:  "chat123",
					Message: "Hi there",
					Role:    AssistantRole,
					Status:  SentStatus,
					Source:  AgentSource,
				},
			},
			expectError: false,
		},
		{
			name: "Empty chat ID",
			setup: func() string {
				return ""
			},
			expected:    []ChatMessage{},
			expectError: false,
		},
		{
			name: "Non-existent chat ID",
			setup: func() string {
				return "nonexistent123"
			},
			expected:    []ChatMessage{},
			expectError: false,
		},
		{
			name: "Chat with special characters in messages",
			setup: func() string {
				chatID := "chat456"
				messages := []ChatMessage{
					{
						ID:        "msg3",
						ChatID:    chatID,
						Message:   "Hello !@#$%^&*()",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "msg3",
					ChatID:  "chat456",
					Message: "Hello !@#$%^&*()",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
			},
			expectError: false,
		},
		{
			name: "Chat with the Unicode messages",
			setup: func() string {
				chatID := "chat789"
				messages := []ChatMessage{
					{
						ID:        "msg4",
						ChatID:    chatID,
						Message:   "你好 👋 Привет",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "msg4",
					ChatID:  "chat789",
					Message: "你好 👋 Привет",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
			},
			expectError: false,
		},
		{
			name: "Chat with large message",
			setup: func() string {
				chatID := "chat101112"
				messages := []ChatMessage{
					{
						ID:        "msg5",
						ChatID:    chatID,
						Message:   strings.Repeat("a", 1000),
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "msg5",
					ChatID:  "chat101112",
					Message: strings.Repeat("a", 1000),
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
			},
			expectError: false,
		},
		{
			name: "SQL injection attempt in chat ID",
			setup: func() string {
				return "chat123'; DROP TABLE chat_messages; --"
			},
			expected:    []ChatMessage{},
			expectError: false,
		},
		{
			name: "Messages ordered by timestamp",
			setup: func() string {
				chatID := "chatOrdered"
				messages := []ChatMessage{
					{
						ID:        "msg6",
						ChatID:    chatID,
						Message:   "Second",
						Role:      UserRole,
						Timestamp: currentTime.Add(time.Minute),
						Status:    SentStatus,
						Source:    UserSource,
					},
					{
						ID:        "msg7",
						ChatID:    chatID,
						Message:   "First",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "msg7",
					ChatID:  "chatOrdered",
					Message: "First",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
				{
					ID:      "msg6",
					ChatID:  "chatOrdered",
					Message: "Second",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
			},
			expectError: false,
		},
		{
			name: "Messages with different statuses",
			setup: func() string {
				chatID := "chatStatus"
				messages := []ChatMessage{
					{
						ID:        "msg8",
						ChatID:    chatID,
						Message:   "Sending",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SendingStatus,
						Source:    UserSource,
					},
					{
						ID:        "msg9",
						ChatID:    chatID,
						Message:   "Error",
						Role:      AssistantRole,
						Timestamp: currentTime.Add(time.Minute),
						Status:    ErrorStatus,
						Source:    AgentSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "msg8",
					ChatID:  "chatStatus",
					Message: "Sending",
					Role:    UserRole,
					Status:  SendingStatus,
					Source:  UserSource,
				},
				{
					ID:      "msg9",
					ChatID:  "chatStatus",
					Message: "Error",
					Role:    AssistantRole,
					Status:  ErrorStatus,
					Source:  AgentSource,
				},
			},
			expectError: false,
		},
		{
			name: "Valid Chat ID with Messages",
			setup: func() string {
				chatID := "valid-chat-123"
				messages := []ChatMessage{
					{
						ID:        "valid-msg-1",
						ChatID:    chatID,
						Message:   "Test message 1",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "valid-msg-1",
					ChatID:  "valid-chat-123",
					Message: "Test message 1",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
			},
			expectError: false,
		},
		{
			name: "Chat ID with Maximum Length",
			setup: func() string {
				chatID := strings.Repeat("a", 255)
				messages := []ChatMessage{
					{
						ID:        "max-length-msg-1",
						ChatID:    chatID,
						Message:   "Max length chat ID",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "max-length-msg-1",
					ChatID:  strings.Repeat("a", 255),
					Message: "Max length chat ID",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
			},
			expectError: false,
		},
		{
			name: "Case Sensitivity",
			setup: func() string {
				chatID := "UPPERCASE-CHAT-ID"
				messages := []ChatMessage{
					{
						ID:        "case-sensitive-msg-1",
						ChatID:    chatID,
						Message:   "Case sensitive test",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return "uppercase-chat-id"
			},
			expected:    []ChatMessage{},
			expectError: false,
		},
		{
			name: "Chat ID with Special Characters",
			setup: func() string {
				chatID := "chat!@#$%^&*()_+"
				messages := []ChatMessage{
					{
						ID:        "special-char-msg-1",
						ChatID:    chatID,
						Message:   "Special characters in chat ID",
						Role:      UserRole,
						Timestamp: currentTime,
						Status:    SentStatus,
						Source:    UserSource,
					},
				}
				for _, msg := range messages {
					TestDB.db.Create(&msg)
				}
				return chatID
			},
			expected: []ChatMessage{
				{
					ID:      "special-char-msg-1",
					ChatID:  "chat!@#$%^&*()_+",
					Message: "Special characters in chat ID",
					Role:    UserRole,
					Status:  SentStatus,
					Source:  UserSource,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			TestDB.db.Exec("DELETE FROM chat_messages")

			chatID := tt.setup()
			messages, err := TestDB.GetChatMessagesForChatID(chatID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expected), len(messages))

				for i, expectedMsg := range tt.expected {
					assert.Equal(t, expectedMsg.ID, messages[i].ID)
					assert.Equal(t, expectedMsg.ChatID, messages[i].ChatID)
					assert.Equal(t, expectedMsg.Message, messages[i].Message)
					assert.Equal(t, expectedMsg.Role, messages[i].Role)
					assert.Equal(t, expectedMsg.Status, messages[i].Status)
					assert.Equal(t, expectedMsg.Source, messages[i].Source)
					if len(expectedMsg.ContextTags) > 0 {
						assert.Equal(t, expectedMsg.ContextTags, messages[i].ContextTags)
					}
				}
			}
		})
	}
}

func TestGetAllChatsForWorkspace(t *testing.T) {
	InitTestDB()

	CleanTestData()

	currentTime := time.Now()

	tests := []struct {
		name        string
		workspaceID string
		setup       func(workspaceID string)
		expected    []Chat
		expectError bool
	}{
		{
			name:        "Valid Workspace ID with Chats",
			workspaceID: "validWorkspaceWithChats",
			setup: func(workspaceID string) {
				TestDB.db.Create(&Chat{ID: "1", WorkspaceID: workspaceID, UpdatedAt: currentTime})
				TestDB.db.Create(&Chat{ID: "2", WorkspaceID: workspaceID, UpdatedAt: currentTime.Add(-time.Hour)})
			},
			expected: []Chat{
				{ID: "1", WorkspaceID: "validWorkspaceWithChats", UpdatedAt: currentTime},
				{ID: "2", WorkspaceID: "validWorkspaceWithChats", UpdatedAt: currentTime.Add(-time.Hour)},
			},
			expectError: false,
		},
		{
			name:        "Valid Workspace ID with No Chats",
			workspaceID: "validWorkspaceNoChats",
			setup:       func(workspaceID string) {},
			expected:    []Chat{},
			expectError: false,
		},
		{
			name:        "Empty Workspace ID",
			workspaceID: "",
			setup:       func(workspaceID string) {},
			expected:    []Chat{},
			expectError: false,
		},
		{
			name:        "Large Number of Chats",
			workspaceID: "workspaceWithManyChats",
			setup: func(workspaceID string) {
				for i := 0; i < 1000; i++ {
					TestDB.db.Create(&Chat{ID: fmt.Sprintf("%d", i), WorkspaceID: workspaceID, UpdatedAt: currentTime})
				}
			},
			expected: func() []Chat {
				chats := make([]Chat, 1000)
				for i := 0; i < 1000; i++ {
					chats[i] = Chat{ID: fmt.Sprintf("%d", i), WorkspaceID: "workspaceWithManyChats", UpdatedAt: currentTime}
				}
				return chats
			}(),
			expectError: false,
		},
		{
			name:        "Special Characters in Workspace ID",
			workspaceID: "special!@#$%^&*()_+{}|:<>?",
			setup: func(workspaceID string) {
				TestDB.db.Create(&Chat{ID: "1", WorkspaceID: workspaceID, UpdatedAt: currentTime})
			},
			expected: []Chat{
				{ID: "1", WorkspaceID: "special!@#$%^&*()_+{}|:<>?", UpdatedAt: currentTime},
			},
			expectError: false,
		},
		{
			name:        "SQL Injection Attempt",
			workspaceID: "1; DROP TABLE chats; --",
			setup:       func(workspaceID string) {},
			expected:    []Chat{},
			expectError: false,
		},
		{
			name:        "Whitespace in Workspace ID",
			workspaceID: "  validWorkspace  ",
			setup: func(workspaceID string) {
				TestDB.db.Create(&Chat{ID: "1", WorkspaceID: "validWorkspace", UpdatedAt: currentTime})
			},
			expected:    []Chat{},
			expectError: false,
		},
		{
			name:        "Non-Existent Workspace ID",
			workspaceID: "nonExistentWorkspace",
			setup:       func(workspaceID string) {},
			expected:    []Chat{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CleanTestData()
			tt.setup(tt.workspaceID)

			chats, err := TestDB.GetAllChatsForWorkspace(tt.workspaceID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expected), len(chats))

				for i, expectedChat := range tt.expected {
					assert.Equal(t, expectedChat.ID, chats[i].ID)
					assert.Equal(t, expectedChat.WorkspaceID, chats[i].WorkspaceID)
					assert.WithinDuration(t, expectedChat.UpdatedAt, chats[i].UpdatedAt, time.Second)
				}
			}
		})
	}
}

func TestUpdateChatMessage(t *testing.T) {
	InitTestDB()
	currentTime := time.Now()

	tests := []struct {
		name        string
		setup       func() ChatMessage
		input       ChatMessage
		expected    ChatMessage
		expectError bool
	}{
		{
			name: "Update All Fields",
			setup: func() ChatMessage {
				msg := ChatMessage{
					ID:        "msg1",
					Message:   "Old Message",
					Status:    "Old Status",
					Role:      UserRole,
					Timestamp: currentTime,
					Source:    UserSource,
				}
				TestDB.db.Create(&msg)
				return msg
			},
			input: ChatMessage{
				ID:      "msg1",
				Message: "New Message",
				Status:  "New Status",
				Role:    AssistantRole,
			},
			expected: ChatMessage{
				ID:      "msg1",
				Message: "New Message",
				Status:  "New Status",
				Role:    AssistantRole,
				Source:  UserSource,
			},
			expectError: false,
		},
		{
			name: "Update Only Message",
			setup: func() ChatMessage {
				msg := ChatMessage{
					ID:        "msg2",
					Message:   "Old Message",
					Status:    SentStatus,
					Role:      UserRole,
					Timestamp: currentTime,
					Source:    UserSource,
				}
				TestDB.db.Create(&msg)
				return msg
			},
			input: ChatMessage{
				ID:      "msg2",
				Message: "New Message",
			},
			expected: ChatMessage{
				ID:      "msg2",
				Message: "New Message",
				Status:  SentStatus,
				Role:    UserRole,
				Source:  UserSource,
			},
			expectError: false,
		},
		{
			name: "Empty ID",
			setup: func() ChatMessage {
				return ChatMessage{}
			},
			input: ChatMessage{
				ID:      "",
				Message: "New Message",
			},
			expected:    ChatMessage{},
			expectError: true,
		},
		{
			name: "Non-Existent ID",
			setup: func() ChatMessage {
				return ChatMessage{}
			},
			input: ChatMessage{
				ID:      "nonexistent",
				Message: "New Message",
			},
			expected:    ChatMessage{},
			expectError: true,
		},
		{
			name: "Large Message Content",
			setup: func() ChatMessage {
				msg := ChatMessage{
					ID:        "msg3",
					Message:   "Old Message",
					Status:    SentStatus,
					Role:      UserRole,
					Timestamp: currentTime,
					Source:    UserSource,
				}
				TestDB.db.Create(&msg)
				return msg
			},
			input: ChatMessage{
				ID:      "msg3",
				Message: strings.Repeat("a", 10000),
			},
			expected: ChatMessage{
				ID:      "msg3",
				Message: strings.Repeat("a", 10000),
				Status:  SentStatus,
				Role:    UserRole,
				Source:  UserSource,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			TestDB.db.Exec("DELETE FROM chat_messages")

			original := tt.setup()
			updatedMsg, err := TestDB.UpdateChatMessage(&tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.ID, updatedMsg.ID)
				assert.Equal(t, tt.expected.Message, updatedMsg.Message)
				assert.Equal(t, tt.expected.Status, updatedMsg.Status)
				assert.Equal(t, tt.expected.Role, updatedMsg.Role)
				assert.Equal(t, tt.expected.Source, updatedMsg.Source)
				assert.True(t, updatedMsg.Timestamp.After(original.Timestamp))
			}
		})
	}
}
