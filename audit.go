package audited

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ContextKey string

const AuditTable = "audit_logs"

func (c ContextKey) String() string {
	return string(c)
}

var (
	ContextKeyEmail = ContextKey("email")
)

// AuditLog represents the audit log model
type AuditLog struct {
	Id            uuid.UUID      `json:"id" gorm:"primaryKey"`
	TableName     string         `json:"table_name"`
	OperationType string         `json:"operation_type"`
	ObjectId      string         `json:"object_id"`
	Data          datatypes.JSON `json:"data"`
	UserId        string         `json:"user_id"`
	CreatedAt     time.Time      `json:"created_at"`
}

// Create method to add create audit log hook
func Create(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.Table == AuditTable || db.Error != nil {
		return
	}

	recordMap, err := getDataBeforeOperation(db)
	if err != nil {
		return
	}
	objId := getKeyFromData("id", recordMap)

	auditLog := &AuditLog{
		TableName:     db.Statement.Schema.Table,
		OperationType: "CREATE",
		ObjectId:      objId,
		Data:          prepareData(recordMap),
		UserId:        getCurrentUser(db.Statement.Context),
	}

	if err := db.Session(&gorm.Session{SkipHooks: true, NewDB: true}).
		Table(AuditTable).
		Create(auditLog).
		Error; err != nil {

		log.Println(fmt.Errorf("error in audit log creation: %s", err.Error()))
		return
	}
}

// Update method to add update audit log hook
func Update(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.Table == AuditTable || db.Error != nil {
		return
	}

	recordMap, err := getDataBeforeOperation(db)
	if err != nil {
		return
	}
	objId := getKeyFromData("id", recordMap)
	auditLog := &AuditLog{
		TableName:     db.Statement.Schema.Table,
		OperationType: "UPDATE",
		ObjectId:      objId,
		Data:          prepareData(recordMap),
		UserId:        getCurrentUser(db.Statement.Context),
	}

	if err := db.Session(&gorm.Session{SkipHooks: true, NewDB: true}).
		Table(AuditTable).
		Create(auditLog).
		Error; err != nil {
		log.Println(fmt.Errorf("error in audit log creation: %s", err.Error()))
		return
	}
}

// Delete method to add delete audit log hook
func Delete(db *gorm.DB) {
	if db.Statement.Schema != nil && db.Statement.Schema.Table == AuditTable || db.Error != nil {
		return
	}

	recordMap, err := getDataBeforeOperation(db)
	if err != nil {
		return
	}
	objId := getKeyFromData("id", recordMap)
	auditLog := &AuditLog{
		TableName:     db.Statement.Schema.Table,
		OperationType: "DELETE",
		ObjectId:      objId,
		Data:          prepareData(recordMap),
		UserId:        getCurrentUser(db.Statement.Context),
	}
	if err := db.Session(&gorm.Session{SkipHooks: true, NewDB: true}).
		Table(AuditTable).
		Create(auditLog).
		Error; err != nil {
		log.Println(fmt.Errorf("error in audit log creation: %s", err.Error()))
		return
	}
}

func getDataBeforeOperation(db *gorm.DB) (map[string]interface{}, error) {
	objMap := map[string]interface{}{}
	if db.Error == nil && !db.DryRun {
		objectType := reflect.TypeOf(db.Statement.ReflectValue.Interface())

		// Create a new instance of the object type
		targetObj := reflect.New(objectType).Interface()

		primaryKeyValue := ""
		value := db.Statement.ReflectValue

		// Check if the value is a struct
		if value.Kind() == reflect.Struct {
			primaryKeyValue = value.FieldByName("Id").String()
		}

		// Fetch the target object separately
		if err := db.Session(&gorm.Session{SkipHooks: true, NewDB: true}).
			Where("id = ?", primaryKeyValue).
			First(&targetObj).
			Error; err != nil {
			log.Println(fmt.Errorf("gorm callback: error while finding target object: %s",
				err.Error()))
			return nil, err
		}

		jsonBytes, err := json.Marshal(targetObj)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(jsonBytes, &objMap); err != nil {
			return objMap, err
		}
	}
	return objMap, nil
}

func RegisterCallbacks(db *gorm.DB) error {
	if err := db.Callback().
		Create().
		After("gorm:create").
		Register("custom_plugin:create_audit_log", Create); err != nil {
		return err
	}
	if err := db.Callback().
		Update().
		After("gorm:update").
		Register("custom_plugin:update_audit_log", Update); err != nil {
		return err
	}

	if err := db.Callback().
		Delete().
		Before("gorm:delete").
		Register("custom_plugin:delete_audit_log", Delete); err != nil {
		return err
	}
	return nil
}

// Sample method to retrieve user currently using the system
func getCurrentUser(ctx context.Context) string {
	if ctx.Value(ContextKeyEmail) == nil {
		log.Println("user not specified in context, please specify user for audit purposes")
		return "ctx-nonspecified"
	}
	return ctx.Value(ContextKeyEmail).(string)
}

func getKeyFromData(key string, data map[string]interface{}) string {
	objId, ok := data[key]
	if !ok {
		return ""
	}
	return objId.(string)
}

func prepareData(data map[string]interface{}) datatypes.JSON {
	dataByte, _ := json.Marshal(&data)
	return dataByte
}
