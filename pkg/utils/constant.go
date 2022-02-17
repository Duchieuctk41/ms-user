package utils

const ADMIN_ROLE = 64

// Status for check valid order item
const (
	STATUS_SUCCESS       = "success"
	STATUS_SKU_NOT_FOUND = "sku_not_found"
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
	TOPIC_CREATE_PO_V2         = "ms-warehouse-management:create_po_v2"
	TOPIC_UPDATE_STOCK         = "ms-warehouse-management:update_stock"
	TOPIC_UPDATE_STOCK_V2      = "ms-warehouse-management:update_stock_v2"
	TOPIC_PROCESS_MISSION      = "finan-loyalty:process_mission"
	TOPIC_SET_USER_GUIDE       = "ms-meta-data:topic_set_user_guide"
	TOPIC_SEND_NOTIFICATION    = "ms-notification-management:send-notification"
)

const CATEGORY_SELL = "cc8244c0-307f-46fd-be96-34e7c36059de"

const (
	ORDER_STATE_WAITING_CONFIRM = "waiting_confirm"
	ORDER_STATE_DELIVERING      = "delivering"
	ORDER_STATE_COMPLETE        = "complete"
	ORDER_STATE_CANCEL          = "cancel"
	ORDER_STATE_UPDATE          = "update"
)

const (
	TOPIC_SEND_EMAIL_ORDER            = "finan-order:send-email-order"
	TOPIC_UPDATE_EMAIL_ORDER_RECENT   = "finan-order:update_email_order_recent"
	TOPIC_UPDATE_SOLD_QUANTITY_CANCEL = "ms-product-management:update_sold_quantity_cancel"
	TOPIC_UPDATE_ORDER_ECOM           = "finan-order:update_order_ecom"
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
	ORDER_EMAIL_UPDATE         = 3
)

const (
	DefaultFromName  = "Sổ bán hàng"
	DefaultFromEmail = "notifications@sobanhang.com"
)

const (
	AVATAR_BUSINESS_DEFAULT = "https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/1d78990d-33ef-4278-94a9-881c7c57d4ae/image/default_avatar_shop.png"
	LINK_IMAGE_RESIZE       = "https://d3hr4eej8cfgwy.cloudfront.net/"
)

const (
	ORDER_COMPLETED      = "order_completed"
	ORDER_CANCELLED      = "order_cancelled"
	FAST_ORDER_COMPLETED = "fast_order_completed"
)

const PO_IN = "in"
const PO_OUT = "out"

// type ms-chat
const (
	NOTI_TYPE_UPDATE_ORDER = "update_order"
)

const TIME_FORMAT_FOR_QUERRY = "2006-01-02 15:04:05"

const (
	COMPLETED_TUTORIAL    = "completed"
	TUTORIAL_CREATE_ORDER = "tutorial_create_order"
)

const (
	TABLE_ORDER      = "order"
	TABLE_ORDER_ITEM = "order_item"
	TABLE_ORDER_ECOM = "order_ecom"
)

const (
	ACTION_CREATE_ORDER        = "create order"
	ACTION_CREATE_ORDER_SELLER = "create order seller"
	ACTION_UPDATE_ORDER        = "update order"

	ACTION_CREATE_ORDER_ITEM           = "create order item"
	ACTION_CREATE_OR_SELECT_ORDER_ITEM = "create or select order item"
	ACTION_DELETE_ORDER_ITEM           = "delete order item"
	ACTION_UPDATE_ORDER_ITEM           = "update order item"
)

const (
	NOTI_CONTENT_WAITING_CONFIRM = "Ting ting! Bạn vừa có đơn hàng mới %v. Hãy xác nhận sẽ giao hàng ngay với khách để chốt đơn"

	NOTI_CONTENT_REMINDER_DELIVERING      = "Bạn đã giao đơn %v - %v cho khách %v chưa? Nhấn để kiểm tra hoặc xác nhận đã giao"
	NOTI_CONTENT_REMINDER_WAITING_CONFIRM = "Đơn hàng %v đang chờ xác nhận. Nhấn để thông báo cho khách đơn đã sẵn sàng giao"
)
