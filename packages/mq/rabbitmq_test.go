package mq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAMQP struct {
	mock.Mock
}

func (m *MockAMQP) Dial(url string) (*amqp.Connection, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*amqp.Connection), args.Error(1)
}

func (m *MockAMQP) Channel(conn *amqp.Connection) (*amqp.Channel, error) {
	args := m.Called(conn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*amqp.Channel), args.Error(1)
}

func TestNewRabbitMQConsumer(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		dialErr error
		wantErr bool
	}{
		{
			name:    "success",
			url:     "amqp://localhost",
			dialErr: nil,
			wantErr: false,
		},
		{
			name:    "dial error",
			url:     "amqp://localhost",
			dialErr: assert.AnError,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAMQP := new(MockAMQP)

			if tt.dialErr == nil {
				mockAMQP.On("Dial", tt.url).Return(&amqp.Connection{}, nil)
				mockAMQP.On("Channel", mock.Anything).Return(&amqp.Channel{}, nil)
			} else {
				mockAMQP.On("Dial", tt.url).Return(nil, tt.dialErr)
			}

			_, err := NewRabbitMQConsumerWithAMQP(tt.url, mockAMQP)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockAMQP.AssertExpectations(t)
		})
	}
}

func TestNewRabbitMQPublisher(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		dialErr error
		wantErr bool
	}{
		{
			name:    "success",
			url:     "amqp://localhost",
			dialErr: nil,
			wantErr: false,
		},
		{
			name:    "dial error",
			url:     "amqp://localhost",
			dialErr: assert.AnError,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAMQP := new(MockAMQP)

			if tt.dialErr == nil {
				mockAMQP.On("Dial", tt.url).Return(&amqp.Connection{}, nil)
				mockAMQP.On("Channel", mock.Anything).Return(&amqp.Channel{}, nil)
			} else {
				mockAMQP.On("Dial", tt.url).Return(nil, tt.dialErr)
			}

			_, err := NewRabbitMQPublisherWithAMQP(tt.url, mockAMQP)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockAMQP.AssertExpectations(t)
		})
	}
}

func TestRabbitMQConsumer_Close(t *testing.T) {
	consumer := &RabbitMQConsumer{
		conn: nil,
		ch:   nil,
	}

	err := consumer.Close()
	assert.NoError(t, err)
}

func TestRabbitMQPublisher_Close(t *testing.T) {
	publisher := &RabbitMQPublisher{
		conn: nil,
		ch:   nil,
	}

	err := publisher.Close()
	assert.NoError(t, err)
}