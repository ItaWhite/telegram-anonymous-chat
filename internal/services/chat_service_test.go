package services

import (
	chatmodels "telegram-anonymous-chat/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	s := NewChatService()
	res, err := s.Start(1, "user1")
	u := s.users[1]
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, chatmodels.StateIdle, u.State)
}

func TestNext_Pairing(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	s.Start(2, "user2")
	u1 := s.users[1]
	u2 := s.users[2]
	res1, err1 := s.Next(1)
	res2, err2 := s.Next(2)
	require.NotNil(t, res1)
	require.NoError(t, err1)
	require.NotNil(t, res2)
	require.NoError(t, err2)
	assert.Equal(t, int64(2), u1.PartnerID)
	assert.Equal(t, int64(1), u2.PartnerID)
	assert.Equal(t, chatmodels.StatePaired, u1.State)
	assert.Equal(t, chatmodels.StatePaired, u2.State)
}

func TestNext_Waiting(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	u := s.users[1]
	s.Next(1)
	res, err := s.Next(1)
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, int64(0), u.PartnerID)
	assert.Equal(t, chatmodels.StateWaiting, u.State)
}

func TestStop_Paired(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	s.Start(2, "user2")
	u1 := s.users[1]
	u2 := s.users[2]
	s.Next(1)
	s.Next(2)
	res, err := s.Stop(1)
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, int64(0), u1.PartnerID)
	assert.Equal(t, int64(0), u2.PartnerID)
	assert.Equal(t, chatmodels.StateIdle, u1.State)
	assert.Equal(t, chatmodels.StateIdle, u2.State)
	assert.True(t, res.ChatEnded)
}

func TestStop_Waiting(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	u := s.users[1]
	s.Next(1)
	res, err := s.Stop(1)
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, chatmodels.StateIdle, u.State)
	assert.False(t, res.ChatEnded)
}

func TestDefault_Paired(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	s.Start(2, "user2")
	s.Next(1)
	s.Next(2)
	res, err := s.Default(1, "hello")
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Messages[0].ChatID)
	assert.Equal(t, "hello", res.Messages[0].Message)
}

func TestDefault_Idle(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	res, err := s.Default(1, "hello")
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Messages[0].ChatID)
	assert.NotEqual(t, "hello", res.Messages[0].Message)
}

func TestChangeRating_Increase(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	u := s.users[1]
	err := s.ChangeRating("like:1")
	require.NoError(t, err)
	assert.Equal(t, 11, u.Rating)
}

func TestChangeRating_Decrease(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	u := s.users[1]
	err := s.ChangeRating("dislike:1")
	require.NoError(t, err)
	assert.Equal(t, 9, u.Rating)
}

func TestManageBlocking_Waiting(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	u := s.users[1]
	s.Next(1)
	res, err := s.ManageBlocking(1)
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, chatmodels.StateIdle, u.State)
}

func TestManageBlocking_Paired(t *testing.T) {
	s := NewChatService()
	s.Start(1, "user1")
	s.Start(2, "user2")
	u1 := s.users[1]
	u2 := s.users[2]
	s.Next(1)
	s.Next(2)
	res, err := s.ManageBlocking(1)
	require.NotNil(t, res)
	require.NoError(t, err)
	assert.Equal(t, int64(0), s.GetPartner(1))
	assert.Equal(t, int64(0), s.GetPartner(2))
	assert.Equal(t, chatmodels.StateIdle, u1.State)
	assert.Equal(t, chatmodels.StateIdle, u2.State)
}
