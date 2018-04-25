package repo

import (
	"crypto/rand"
	"encoding/json"
	//"errors"
	"fmt"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	"time"
)

// Notifier is an interface which is used to send data to the frontend
type Notifier interface {
	// GetID returns the unique string identifier for the Notifier and is used to
	// uniquely persist the Notifier in the DB. Some Notifiers are not persisted.
	// Until we can represent this as part of the interface, the Notifiers which
	// do not get persisted can safely return an empty string. Notifiers which are
	// persisted and return a non-unique GetID() string will eventually fail the DB's
	// uniqueness contraints during runtime.
	GetID() string

	// GetType returns the type as a NotificationType
	GetType() NotificationType

	// GetSMTPTitleAndBody returns the title and body strings to be used
	// in any notification content. The bool can return false to bypass the
	// SMTP notification for this Notifier.
	GetSMTPTitleAndBody() (string, string, bool)

	// Data returns the marhsalled []byte suitable for transmission to the client
	// over a websocket connection
	Data() ([]byte, error)
}

// NewNotification is a helper that returns a properly instantiated *Notification
func NewNotification(n Notifier, createdAt time.Time, isRead bool) *Notification {
	return &Notification{
		ID:           n.GetID(),
		CreatedAt:    createdAt,
		IsRead:       isRead,
		NotifierData: n,
		NotifierType: n.GetType(),
	}
}

// Notification represents both a record from the Notifications Datastore as well
// as an unmarshalling envelope for the Notifier interface field NotifierData.
// NOTE: Only ID, NotifierData and NotifierType fields are valid in both contexts. This is
// because (*Notification).MarshalJSON only wraps the NotifierData field. NotifierData
// describes ID and NotifierType and will also be valid when unmarshalled.
// TODO: Ecapsulate the whole Notification struct inside of MarshalJSON and update persisted
// serializations to match in the Notifications Datastore
type Notification struct {
	ID           string           `json:"-"`
	CreatedAt    time.Time        `json:"timestamp"`
	IsRead       bool             `json:"read"`
	NotifierData Notifier         `json:"notification"`
	NotifierType NotificationType `json:"-"`
}

func (n *Notification) GetID() string         { return n.ID }
func (n *Notification) GetTypeString() string { return string(n.GetType()) }
func (n *Notification) GetType() NotificationType {
	if string(n.NotifierType) == "" {
		n.NotifierType = n.NotifierData.GetType()
	}
	return n.NotifierType
}
func (n *Notification) GetUnixCreatedAt() int { return int(n.CreatedAt.Unix()) }
func (n *Notification) GetSMTPTitleAndBody() (string, string, bool) {
	return n.NotifierData.GetSMTPTitleAndBody()
}
func (n *Notification) Data() ([]byte, error) { return json.MarshalIndent(n, "", "    ") }

type notificationTransporter struct {
	CreatedAt    time.Time        `json:"timestamp"`
	IsRead       bool             `json:"read"`
	NotifierData json.RawMessage  `json:"notification"`
	NotifierType NotificationType `json:"type"`
}

func (n *Notification) UnmarshalJSON(data []byte) error {
	var payload notificationTransporter
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	switch payload.NotifierType {
	case NotifierTypeSaleAgedFourtyFiveDays:
		var notifier = SaleAgingNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier

	case NotifierTypeDisputeAgedZeroDays:
		fallthrough
	case NotifierTypeDisputeAgedFifteenDays:
		fallthrough
	case NotifierTypeDisputeAgedFourtyDays:
		fallthrough
	case NotifierTypeDisputeAgedFourtyFourDays:
		fallthrough
	case NotifierTypeDisputeAgedFourtyFiveDays:
		var notifier = DisputeAgingNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier

	case NotifierTypePurchaseAgedZeroDays:
		fallthrough
	case NotifierTypePurchaseAgedFifteenDays:
		fallthrough
	case NotifierTypePurchaseAgedFourtyDays:
		fallthrough
	case NotifierTypePurchaseAgedFourtyFourDays:
		fallthrough
	case NotifierTypePurchaseAgedFourtyFiveDays:
		var notifier = PurchaseAgingNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeCompletionNotification:
		var notifier = CompletionNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeDisputeAcceptedNotification:
		var notifier = DisputeAcceptedNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeDisputeCloseNotification:
		var notifier = DisputeCloseNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeDisputeOpenNotification:
		var notifier = DisputeOpenNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeDisputeUpdateNotification:
		var notifier = DisputeUpdateNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeFollowNotification:
		var notifier = FollowNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeFulfillmentNotification:
		var notifier = FulfillmentNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeModeratorAddNotification:
		var notifier = ModeratorAddNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeModeratorRemoveNotification:
		var notifier = ModeratorRemoveNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeOrderCancelNotification:
		var notifier = OrderCancelNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeOrderConfirmationNotification:
		var notifier = OrderConfirmationNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeOrderDeclinedNotification:
		var notifier = OrderDeclinedNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeOrderNewNotification:
		var notifier = OrderNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypePaymentNotification:
		var notifier = PaymentNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeProcessingErrorNotification:
		var notifier = ProcessingErrorNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeRefundNotification:
		var notifier = RefundNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	case NotifierTypeUnfollowNotification:
		var notifier = UnfollowNotification{}
		if err := json.Unmarshal(data, &notifier); err != nil {
			return err
		}
		n.NotifierData = notifier
	default:
		return fmt.Errorf("unknown notifier type: %t", payload.NotifierType)
	}

	n.ID = n.NotifierData.GetID()
	n.NotifierType = payload.NotifierType
	return nil
}

type Thumbnail struct {
	Tiny  string `json:"tiny"`
	Small string `json:"small"`
}

type notificationWrapper struct {
	Notification Notifier `json:"notification"`
}

type messageWrapper struct {
	Message Notifier `json:"message"`
}

type walletWrapper struct {
	Message Notifier `json:"wallet"`
}

type messageReadWrapper struct {
	MessageRead Notifier `json:"messageRead"`
}

type messageTypingWrapper struct {
	MessageRead Notifier `json:"messageTyping"`
}

type OrderNotification struct {
	ID          string           `json:"notificationId"`
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	BuyerID     string           `json:"buyerId"`
	BuyerHandle string           `json:"buyerHandle"`
	Thumbnail   Thumbnail        `json:"thumbnail"`
	OrderId     string           `json:"orderId"`
	Slug        string           `json:"slug"`
}

func (n OrderNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n OrderNotification) GetID() string             { return n.ID }
func (n OrderNotification) GetType() NotificationType { return NotifierTypeOrderNewNotification }
func (n OrderNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "You received an order \"%s\".\n\nOrder ID: %s\nBuyer: %s\nThumbnail: %s\n"
	return "Order received", fmt.Sprintf(form, n.Title, n.OrderId, n.getBuyerID(), n.Thumbnail.Small), true
}
func (n OrderNotification) getBuyerID() string {
	if n.BuyerHandle != "" {
		return n.BuyerHandle
	}
	return n.BuyerID
}

type PaymentNotification struct {
	ID           string           `json:"notificationId"`
	Type         NotificationType `json:"type"`
	OrderId      string           `json:"orderId"`
	FundingTotal uint64           `json:"fundingTotal"`
}

func (n PaymentNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n PaymentNotification) GetID() string             { return n.ID }
func (n PaymentNotification) GetType() NotificationType { return NotifierTypePaymentNotification }
func (n PaymentNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Payment for order \"%s\" received (total %d)."
	return "Payment received", fmt.Sprintf(form, n.OrderId, n.FundingTotal), true
}

type OrderConfirmationNotification struct {
	ID           string           `json:"notificationId"`
	Type         NotificationType `json:"type"`
	OrderId      string           `json:"orderId"`
	Thumbnail    Thumbnail        `json:"thumbnail"`
	VendorHandle string           `json:"vendorHandle"`
	VendorID     string           `json:"vendorId"`
}

func (n OrderConfirmationNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n OrderConfirmationNotification) GetID() string { return n.ID }
func (n OrderConfirmationNotification) GetType() NotificationType {
	return NotifierTypeOrderConfirmationNotification
}
func (n OrderConfirmationNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Order \"%s\" has been confirmed."
	return "Order confirmed", fmt.Sprintf(form, n.OrderId), true
}

type OrderDeclinedNotification struct {
	ID           string           `json:"notificationId"`
	Type         NotificationType `json:"type"`
	OrderId      string           `json:"orderId"`
	Thumbnail    Thumbnail        `json:"thumbnail"`
	VendorHandle string           `json:"vendorHandle"`
	VendorID     string           `json:"vendorId"`
}

func (n OrderDeclinedNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n OrderDeclinedNotification) GetID() string { return n.ID }
func (n OrderDeclinedNotification) GetType() NotificationType {
	return NotifierTypeOrderDeclinedNotification
}
func (n OrderDeclinedNotification) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type OrderCancelNotification struct {
	ID          string           `json:"notificationId"`
	Type        NotificationType `json:"type"`
	OrderId     string           `json:"orderId"`
	Thumbnail   Thumbnail        `json:"thumbnail"`
	BuyerHandle string           `json:"buyerHandle"`
	BuyerID     string           `json:"buyerId"`
}

func (n OrderCancelNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n OrderCancelNotification) GetID() string { return n.ID }
func (n OrderCancelNotification) GetType() NotificationType {
	return NotifierTypeOrderCancelNotification
}
func (n OrderCancelNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Order \"%s\" has been cancelled."
	return "Order cancelled", fmt.Sprintf(form, n.OrderId), true
}

type RefundNotification struct {
	ID           string           `json:"notificationId"`
	Type         NotificationType `json:"type"`
	OrderId      string           `json:"orderId"`
	Thumbnail    Thumbnail        `json:"thumbnail"`
	VendorHandle string           `json:"vendorHandle"`
	VendorID     string           `json:"vendorId"`
}

func (n RefundNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n RefundNotification) GetID() string             { return n.ID }
func (n RefundNotification) GetType() NotificationType { return NotifierTypeRefundNotification }
func (n RefundNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Payment refund for order \"%s\" received."
	return "Payment refunded", fmt.Sprintf(form, n.OrderId), true
}

type FulfillmentNotification struct {
	ID           string           `json:"notificationId"`
	Type         NotificationType `json:"type"`
	OrderId      string           `json:"orderId"`
	Thumbnail    Thumbnail        `json:"thumbnail"`
	VendorHandle string           `json:"vendorHandle"`
	VendorID     string           `json:"vendorId"`
}

func (n FulfillmentNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n FulfillmentNotification) GetID() string { return n.ID }
func (n FulfillmentNotification) GetType() NotificationType {
	return NotifierTypeFulfillmentNotification
}
func (n FulfillmentNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Order \"%s\" was marked as fulfilled."
	return "Order fulfilled", fmt.Sprintf(form, n.OrderId), true
}

type ProcessingErrorNotification struct {
	ID           string           `json:"notificationId"`
	Type         NotificationType `json:"type"`
	OrderId      string           `json:"orderId"`
	Thumbnail    Thumbnail        `json:"thumbnail"`
	VendorHandle string           `json:"vendorHandle"`
	VendorID     string           `json:"vendorId"`
}

func (n ProcessingErrorNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n ProcessingErrorNotification) GetID() string { return n.ID }
func (n ProcessingErrorNotification) GetType() NotificationType {
	return NotifierTypeProcessingErrorNotification
}
func (n ProcessingErrorNotification) GetSMTPTitleAndBody() (string, string, bool) {
	return "", "", false
}

type CompletionNotification struct {
	ID          string           `json:"notificationId"`
	Type        NotificationType `json:"type"`
	OrderId     string           `json:"orderId"`
	Thumbnail   Thumbnail        `json:"thumbnail"`
	BuyerHandle string           `json:"buyerHandle"`
	BuyerID     string           `json:"buyerId"`
}

func (n CompletionNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n CompletionNotification) GetID() string             { return n.ID }
func (n CompletionNotification) GetType() NotificationType { return NotifierTypeCompletionNotification }
func (n CompletionNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Order \"%s\" was marked as completed."
	return "Order completed", fmt.Sprintf(form, n.OrderId), true
}

type DisputeOpenNotification struct {
	ID             string           `json:"notificationId"`
	Type           NotificationType `json:"type"`
	OrderId        string           `json:"orderId"`
	Thumbnail      Thumbnail        `json:"thumbnail"`
	DisputerID     string           `json:"disputerId"`
	DisputerHandle string           `json:"disputerHandle"`
	DisputeeID     string           `json:"disputeeId"`
	DisputeeHandle string           `json:"disputeeHandle"`
	Buyer          string           `json:"buyer"`
}

func (n DisputeOpenNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n DisputeOpenNotification) GetID() string { return n.ID }
func (n DisputeOpenNotification) GetType() NotificationType {
	return NotifierTypeDisputeOpenNotification
}
func (n DisputeOpenNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Dispute around order \"%s\" was opened."
	return "Dispute opened", fmt.Sprintf(form, n.OrderId), true
}

type DisputeUpdateNotification struct {
	ID             string           `json:"notificationId"`
	Type           NotificationType `json:"type"`
	OrderId        string           `json:"orderId"`
	Thumbnail      Thumbnail        `json:"thumbnail"`
	DisputerID     string           `json:"disputerId"`
	DisputerHandle string           `json:"disputerHandle"`
	DisputeeID     string           `json:"disputeeId"`
	DisputeeHandle string           `json:"disputeeHandle"`
	Buyer          string           `json:"buyer"`
}

func (n DisputeUpdateNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n DisputeUpdateNotification) GetID() string { return n.ID }
func (n DisputeUpdateNotification) GetType() NotificationType {
	return NotifierTypeDisputeUpdateNotification
}
func (n DisputeUpdateNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Dispute around order \"%s\" was updated."
	return "Dispute updated", fmt.Sprintf(form, n.OrderId), true
}

type DisputeCloseNotification struct {
	ID               string           `json:"notificationId"`
	Type             NotificationType `json:"type"`
	OrderId          string           `json:"orderId"`
	Thumbnail        Thumbnail        `json:"thumbnail"`
	OtherPartyID     string           `json:"otherPartyId"`
	OtherPartyHandle string           `json:"otherPartyHandle"`
	Buyer            string           `json:"buyer"`
}

func (n DisputeCloseNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n DisputeCloseNotification) GetID() string { return n.ID }
func (n DisputeCloseNotification) GetType() NotificationType {
	return NotifierTypeDisputeCloseNotification
}
func (n DisputeCloseNotification) GetSMTPTitleAndBody() (string, string, bool) {
	form := "Dispute around order \"%s\" was closed."
	return "Dispute closed", fmt.Sprintf(form, n.OrderId), true
}

type DisputeAcceptedNotification struct {
	ID               string           `json:"notificationId"`
	Type             NotificationType `json:"type"`
	OrderId          string           `json:"orderId"`
	Thumbnail        Thumbnail        `json:"thumbnail"`
	OherPartyID      string           `json:"otherPartyId"`
	OtherPartyHandle string           `json:"otherPartyHandle"`
	Buyer            string           `json:"buyer"`
}

func (n DisputeAcceptedNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n DisputeAcceptedNotification) GetID() string { return n.ID }
func (n DisputeAcceptedNotification) GetType() NotificationType {
	return NotifierTypeDisputeAcceptedNotification
}
func (n DisputeAcceptedNotification) GetSMTPTitleAndBody() (string, string, bool) {
	return "", "", false
}

type FollowNotification struct {
	ID     string           `json:"notificationId"`
	Type   NotificationType `json:"type"`
	PeerId string           `json:"peerId"`
}

func (n FollowNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n FollowNotification) GetID() string { return n.ID }
func (n FollowNotification) GetType() NotificationType {
	return NotifierTypeFollowNotification
}
func (n FollowNotification) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type UnfollowNotification struct {
	ID     string           `json:"notificationId"`
	Type   NotificationType `json:"type"`
	PeerId string           `json:"peerId"`
}

func (n UnfollowNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n UnfollowNotification) GetID() string                               { return n.ID }
func (n UnfollowNotification) GetType() NotificationType                   { return NotifierTypeUnfollowNotification }
func (n UnfollowNotification) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type ModeratorAddNotification struct {
	ID     string           `json:"notificationId"`
	Type   NotificationType `json:"type"`
	PeerId string           `json:"peerId"`
}

func (n ModeratorAddNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n ModeratorAddNotification) GetID() string { return n.ID }
func (n ModeratorAddNotification) GetType() NotificationType {
	return NotifierTypeModeratorAddNotification
}
func (n ModeratorAddNotification) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type ModeratorRemoveNotification struct {
	ID     string           `json:"notificationId"`
	Type   NotificationType `json:"type"`
	PeerId string           `json:"peerId"`
}

func (n ModeratorRemoveNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n ModeratorRemoveNotification) GetID() string { return n.ID }
func (n ModeratorRemoveNotification) GetType() NotificationType {
	return NotifierTypeModeratorRemoveNotification
}
func (n ModeratorRemoveNotification) GetSMTPTitleAndBody() (string, string, bool) {
	return "", "", false
}

type StatusNotification struct {
	Status string `json:"status"`
}

func (n StatusNotification) Data() ([]byte, error)                       { return json.MarshalIndent(n, "", "    ") }
func (n StatusNotification) GetID() string                               { return "" } // Not persisted, ID is ignored
func (n StatusNotification) GetType() NotificationType                   { return NotifierTypeStatusUpdateNotification }
func (n StatusNotification) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type ChatMessage struct {
	MessageId string    `json:"messageId"`
	PeerId    string    `json:"peerId"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	Outgoing  bool      `json:"outgoing"`
	Timestamp time.Time `json:"timestamp"`
}

func (n ChatMessage) Data() ([]byte, error)                       { return json.MarshalIndent(messageWrapper{n}, "", "    ") }
func (n ChatMessage) GetID() string                               { return "" } // Not persisted, ID is ignored
func (n ChatMessage) GetType() NotificationType                   { return NotifierTypeChatMessage }
func (n ChatMessage) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type ChatRead struct {
	MessageId string `json:"messageId"`
	PeerId    string `json:"peerId"`
	Subject   string `json:"subject"`
}

func (n ChatRead) Data() ([]byte, error)                       { return json.MarshalIndent(messageReadWrapper{n}, "", "    ") }
func (n ChatRead) GetID() string                               { return "" } // Not persisted, ID is ignored
func (n ChatRead) GetType() NotificationType                   { return NotifierTypeChatRead }
func (n ChatRead) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type ChatTyping struct {
	MessageId string `json:"messageId"`
	PeerId    string `json:"peerId"`
	Subject   string `json:"subject"`
}

func (n ChatTyping) Data() ([]byte, error) {
	return json.MarshalIndent(messageTypingWrapper{n}, "", "    ")
}
func (n ChatTyping) GetID() string                               { return "" } // Not persisted, ID is ignored
func (n ChatTyping) GetType() NotificationType                   { return NotifierTypeChatTyping }
func (n ChatTyping) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type IncomingTransaction struct {
	Txid          string    `json:"txid"`
	Value         int64     `json:"value"`
	Address       string    `json:"address"`
	Status        string    `json:"status"`
	Memo          string    `json:"memo"`
	Timestamp     time.Time `json:"timestamp"`
	Confirmations int32     `json:"confirmations"`
	OrderId       string    `json:"orderId"`
	Thumbnail     string    `json:"thumbnail"`
	Height        int32     `json:"height"`
	CanBumpFee    bool      `json:"canBumpFee"`
}

func (n IncomingTransaction) Data() ([]byte, error) {
	return json.MarshalIndent(walletWrapper{n}, "", "    ")
}
func (n IncomingTransaction) GetID() string                               { return "" } // Not persisted, ID is ignored
func (n IncomingTransaction) GetType() NotificationType                   { return NotifierTypeIncomingTransaction }
func (n IncomingTransaction) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

// SaleAgingNotification represents a notification about a sale
// which will soon be unable to dispute. The Type indicates the age of the
// purchase and OrderID references the purchases orderID in the database schema
type SaleAgingNotification struct {
	ID      string           `json:"notificationId"`
	Type    NotificationType `json:"type"`
	OrderID string           `json:"purchaseOrderId"`
}

func (n SaleAgingNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n SaleAgingNotification) GetID() string             { return n.ID }
func (n SaleAgingNotification) GetType() NotificationType { return n.Type }
func (n SaleAgingNotification) GetSMTPTitleAndBody() (string, string, bool) {
	return "", "", false
}

// PurchaseAgingNotification represents a notification about a sale
// which will soon be unable to dispute. The Type indicates the age of the
// sale and OrderID references the sale's orderID in the database schema
type PurchaseAgingNotification struct {
	ID      string           `json:"notificationId"`
	Type    NotificationType `json:"type"`
	OrderID string           `json:"saleOrderId"`
}

func (n PurchaseAgingNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n PurchaseAgingNotification) GetID() string             { return n.ID }
func (n PurchaseAgingNotification) GetType() NotificationType { return n.Type }
func (n PurchaseAgingNotification) GetSMTPTitleAndBody() (string, string, bool) {
	return "", "", false
}

// DisputeAgingNotification represents a notification about an open dispute
// which will soon be expired and automatically resolved. The Type indicates
// the age of the dispute case and the CaseID references the cases caseID
// in the database schema
type DisputeAgingNotification struct {
	ID     string           `json:"notificationId"`
	Type   NotificationType `json:"type"`
	CaseID string           `json:"disputeCaseId"`
}

func (n DisputeAgingNotification) Data() ([]byte, error) {
	return json.MarshalIndent(notificationWrapper{n}, "", "    ")
}
func (n DisputeAgingNotification) GetID() string                               { return n.ID }
func (n DisputeAgingNotification) GetType() NotificationType                   { return n.Type }
func (n DisputeAgingNotification) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

type TestNotification struct{}

func (TestNotification) Data() ([]byte, error) {
	return json.MarshalIndent(TestNotification{}, "", "    ")
}
func (TestNotification) GetID() string             { return "" } // Not persisted, ID is ignored
func (TestNotification) GetType() NotificationType { return NotifierTypeTestNotification }
func (TestNotification) GetSMTPTitleAndBody() (string, string, bool) {
	return "Test Notification Head", "Test Notification Body", true
}

// PremarshalledNotifier is a hack to allow []byte data to be transfered through
// the Notifier interface without having to do things the right way. You should not
// be using this and should prefer to use an existing Notifier struct or create
// a new one following the pattern of the TestNotification
type PremarshalledNotifier struct {
	Payload []byte
}

func (p PremarshalledNotifier) Data() ([]byte, error)                       { return p.Payload, nil }
func (n PremarshalledNotifier) GetID() string                               { return "" } // Not persisted, ID is ignored
func (p PremarshalledNotifier) GetType() NotificationType                   { return NotifierTypePremarshalledNotifier }
func (p PremarshalledNotifier) GetSMTPTitleAndBody() (string, string, bool) { return "", "", false }

func NewNotificationID() string {
	b := make([]byte, 32)
	rand.Read(b)
	encoded, _ := mh.Encode(b, mh.SHA2_256)
	nId, _ := mh.Cast(encoded)
	return nId.B58String()
}
