// Copyright 2018-2019 go-m3ua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package m3ua

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/wmnsk/go-m3ua/messages"
	"github.com/wmnsk/go-m3ua/messages/params"
)

// Error definitions.
var (
	ErrSCTPNotAlive        = errors.New("SCTP is no longer alive")
	ErrInvalidState        = errors.New("invalid state")
	ErrNotEstablished      = errors.New("M3UA Conn not established")
	ErrFailedToEstablish   = errors.New("failed to establish M3UA Conn")
	ErrTimeout             = errors.New("timed out")
	ErrHeartbeatExpired    = errors.New("heartbeat timer expired")
	ErrFailedToPeelOff     = errors.New("failed to peel off Protocol Data")
	ErrFailedToWriteSignal = errors.New("failed to write signal")
)

// ErrInvalidVersion is used if a message with an unsupported version is received.
type ErrInvalidVersion struct {
	Ver uint8
}

// NewErrInvalidVersion creates ErrInvalidVersion.
func NewErrInvalidVersion(ver uint8) *ErrInvalidVersion {
	return &ErrInvalidVersion{Ver: ver}
}

// Error returns error string with violating version.
func (e *ErrInvalidVersion) Error() string {
	return fmt.Sprintf("invalid version: %d", e.Ver)
}

// ErrUnsupportedClass is used if a message with an unexpected or
// unsupported Message Class is received.
type ErrUnsupportedClass struct {
	Msg messages.M3UA
}

// NewErrUnsupportedClass creates ErrUnsupportedClass
func NewErrUnsupportedClass(msg messages.M3UA) *ErrUnsupportedClass {
	return &ErrUnsupportedClass{Msg: msg}
}

// Error returns error string with message class.
func (e *ErrUnsupportedClass) Error() string {
	return fmt.Sprintf("message class unsupported. class: %s", e.Msg.MessageClassName())
}

func (e *ErrUnsupportedClass) first40Octets() []byte {
	b, err := e.Msg.Serialize()
	if err != nil {
		return nil
	}
	if len(b) < 40 {
		return b
	}

	return b[:40]
}

// ErrUnsupportedMessage is used if a message with an
// unexpected or unsupported Message Type is received.
type ErrUnsupportedMessage struct {
	Msg messages.M3UA
}

// NewErrUnsupportedMessage creates ErrUnsupportedMessage
func NewErrUnsupportedMessage(msg messages.M3UA) *ErrUnsupportedMessage {
	return &ErrUnsupportedMessage{Msg: msg}
}

// Error returns error string with message class and type.
func (e *ErrUnsupportedMessage) Error() string {
	return fmt.Sprintf("message unsupported. class: %s, type: %s", e.Msg.MessageClassName(), e.Msg.MessageTypeName())
}

func (e *ErrUnsupportedMessage) first40Octets() []byte {
	b, err := e.Msg.Serialize()
	if err != nil {
		return nil
	}
	if len(b) < 40 {
		return b
	}

	return b[:40]
}

// ErrUnexpectedMessage is used if a defined and recognized message is received
// that is not expected in the current state (in some cases, the ASP may optionally
// silently discard the message and not send an Error message).
type ErrUnexpectedMessage struct {
	Msg messages.M3UA
}

// NewErrUnexpectedMessage creates ErrUnexpectedMessage
func NewErrUnexpectedMessage(msg messages.M3UA) *ErrUnexpectedMessage {
	return &ErrUnexpectedMessage{Msg: msg}
}

// Error returns error string with message class and type.
func (e *ErrUnexpectedMessage) Error() string {
	return fmt.Sprintf("unexpected message. class: %s, type: %s", e.Msg.MessageClassName(), e.Msg.MessageTypeName())
}

// ErrInvalidSCTPStreamID is used if a message is received on an unexpected SCTP stream.
type ErrInvalidSCTPStreamID struct {
	ID uint16
}

// NewErrInvalidSCTPStreamID creates ErrInvalidSCTPStreamID
func NewErrInvalidSCTPStreamID(id uint16) *ErrInvalidSCTPStreamID {
	return &ErrInvalidSCTPStreamID{ID: id}
}

// Error returns error string with violating stream ID.
func (e *ErrInvalidSCTPStreamID) Error() string {
	return fmt.Sprintf("invalid SCTP Stream ID: %d", e.ID)
}

// ErrAspIDRequired is used by an SGP in response to an ASP Up message that
// does not contain an ASP Identifier parameter when the SGP requires one..
type ErrAspIDRequired struct{}

// NewErrAspIDRequired creates ErrAspIDRequired
func NewErrAspIDRequired() *ErrAspIDRequired {
	return &ErrAspIDRequired{}
}

// Error returns error string.
func (e *ErrAspIDRequired) Error() string {
	return fmt.Sprintf("ASP ID required")
}

func (c *Conn) handleErrors(e error) error {
	var res messages.M3UA
	switch err := e.(type) {
	case *ErrInvalidVersion:
		res = messages.NewError(
			params.NewErrorCode(params.ErrInvalidVersion),
			nil, nil, nil, nil,
		)
	case *ErrUnsupportedClass:
		res = messages.NewError(
			params.NewErrorCode(params.ErrUnsupportedMessageClass),
			nil, nil, nil,
			params.NewDiagnosticInformation(err.first40Octets()),
		)
	case *ErrUnsupportedMessage:
		res = messages.NewError(
			params.NewErrorCode(params.ErrUnsupportedMessageType),
			nil, nil, nil,
			params.NewDiagnosticInformation(err.first40Octets()),
		)
	case *ErrUnexpectedMessage:
		res = messages.NewError(
			params.NewErrorCode(params.ErrUnexpectedMessage),
			c.cfg.RoutingContexts,
			c.cfg.NetworkAppearance,
			params.NewAffectedPointCode(
				c.cfg.OriginatingPointCode,
			),
			nil,
		)
	case *ErrInvalidSCTPStreamID:
		res = messages.NewError(
			params.NewErrorCode(params.ErrInvalidStreamIdentifier),
			nil, nil, nil, nil,
		)
	case *ErrAspIDRequired:
		res = messages.NewError(
			params.NewErrorCode(params.ErrAspIdentifierRequired),
			nil, nil, nil, nil,
		)
	default:
		return e
	}

	if _, err := c.WriteSignal(res); err != nil {
		return err
	}
	return nil
}
