package utils

const ADMIN_ROLE = 64

// Status for check valid order item
const (
	STATUS_SUCCESS      = "success"
	STATUS_OUT_OF_STOCK = "out_of_stock"
)

const (
	SELLER_CREATE_METHOD = "seller"
	BUYER_CREATE_METHOD  = "buyer"
)

//Delivery Method
const (
	DELIVERY_METHOD_SELLER_DELIVERY = "seller_delivery"
	DELIVERY_METHOD_BUYER_PICK_UP   = "buyer_pick_up"
)

// Topic consumer
const (
	TOPIC_UPDATE_CUSTOM_FIELDS = "ms-business-management:custom_fields"
	TOPIC_UPDATE_SOLD_QUANTITY = "ms-product-management:update_sold_quantity"
	TOPIC_CREATE_PO            = "ms-warehouse-management:create_po"
	TOPIC_UPDATE_STOCK         = "ms-warehouse-management:update_stock"
)

const CATEGORY_SELL = "cc8244c0-307f-46fd-be96-34e7c36059de"

const (
	ORDER_STATE_WAITING_CONFIRM = "waiting_confirm"
	ORDER_STATE_DELIVERING      = "delivering"
	ORDER_STATE_COMPLETE        = "complete"
	ORDER_STATE_CANCEL          = "cancel"
)

const (
	TOPIC_SEND_EMAIL_ORDER = "finan-order:send-email-order"
)

// Some type mess
const (
	MESS_TYPE_UPDATE_ORDER = "update_order"
	MESS_TYPE_SHOW_INVOICE = "invoice"
)

// Some message content from system when order updated
const (
	MESS_ORDER_UPDATE_DETAIL   = "Đơn hàng %v của bạn đã thay đổi. Vui lòng xác nhận lại đơn hàng và liên hệ với cửa hàng nếu có thắc mắc"
	MESS_ORDER_WAITING_CONFIRM = "Đơn hàng %v đã được đặt thành công"
	MESS_ORDER_DELIVERING      = "Đơn hàng %v đã được xác nhận từ người bán"
	MESS_ORDER_COMPLETED       = "Đơn hàng %v đã được giao thành công"
	MESS_ORDER_CANCELED        = "Đơn hàng %v đã bị hủy"
)

const (
	NOTIFICATION_ENTITY_KEY_ORDER                = "order"
	NOTIFICATION_DEEP_LINK_ORDER_WAITING_CONFIRM = "orderManagement_1"
	NOTIFICATION_DEEP_LINK_ORDER_DELIVERING      = "orderManagement_2"
	NOTIFICATION_DEEP_LINK_ORDER_COMPLETE        = "orderManagement_3"
)

//Order send email state
const (
	SEND_EMAIL_WAITING_CONFIRM = 3
	SEND_EMAIL_DELIVERING      = 3
	SEND_EMAIL_COMPLETE        = 3
	SEND_EMAIL_CANCEL          = 3
)

const (
	DefaultFromName  = "Sổ bán hàng"
	DefaultFromEmail = "auto-reply@sobanhang.com"
)
