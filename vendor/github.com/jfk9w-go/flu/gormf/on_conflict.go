package gormf

import (
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// OnConflictClause generates ON CONFLICT clause containing columns tagged with "setting" in the struct of "entity".
func OnConflictClause(entity interface{}, setting string, updateAll bool, doUpdates clause.Set) *clause.OnConflict {
	settingColumns := CollectTaggedColumns(entity, setting)
	if len(settingColumns) == 0 {
		return nil
	}

	columns := make([]clause.Column, 0, len(settingColumns))
	for column := range settingColumns {
		columns = append(columns, clause.Column{Name: column})
	}

	return &clause.OnConflict{
		Columns:   columns,
		UpdateAll: updateAll,
		DoUpdates: doUpdates,
	}
}

var namingStrategy schema.NamingStrategy

// CollectTaggedColumns collects all columns from this entity tagged with a setting
// from this entity and all its embedded structs.
func CollectTaggedColumns(entity interface{}, setting string) map[string]string {
	setting = strings.ToUpper(setting)
	var entityType reflect.Type
	var ok bool
	if entityType, ok = entity.(reflect.Type); !ok {
		entityType = reflect.TypeOf(entity)
	}

	switch entityType.Kind() {
	case reflect.Slice, reflect.Array, reflect.Ptr:
		entityType = entityType.Elem()
	}

	taggedColumns := make(map[string]string)
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if tag, ok := field.Tag.Lookup("gorm"); ok {
			tagSettings := schema.ParseTagSetting(tag, ";")
			if _, ok := tagSettings["EMBEDDED"]; ok {
				for k, v := range CollectTaggedColumns(field.Type, setting) {
					taggedColumns[k] = v
				}
			}

			value, ok := tagSettings[setting]
			if !ok {
				continue
			}

			columnName, ok := tagSettings["COLUMN"]
			if !ok {
				columnName = namingStrategy.ColumnName("", field.Name)
			}

			taggedColumns[columnName] = value
		}
	}

	return taggedColumns
}
