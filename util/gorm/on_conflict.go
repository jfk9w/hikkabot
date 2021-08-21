package gorm

import (
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func OnConflictClause(entity interface{}, setting string, updateAll bool, doUpdates clause.Set) *clause.OnConflict {
	settingColumns := CollectTaggedColumns(entity, setting)
	if len(settingColumns) == 0 {
		return nil
	}

	columns := make([]clause.Column, len(settingColumns))
	for i, column := range settingColumns {
		columns[i] = clause.Column{Name: column}
	}

	return &clause.OnConflict{
		Columns:   columns,
		UpdateAll: updateAll,
		DoUpdates: doUpdates,
	}
}

var namingStrategy schema.NamingStrategy

func CollectTaggedColumns(entity interface{}, setting string) []string {
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

	taggedColumns := make([]string, 0)
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if tag, ok := field.Tag.Lookup("gorm"); ok {
			tagSettings := schema.ParseTagSetting(tag, ";")
			if _, ok := tagSettings["embedded"]; ok {
				taggedColumns = append(taggedColumns, CollectTaggedColumns(field.Type, setting)...)
			}

			if _, ok := tagSettings[setting]; !ok {
				continue
			}

			columnName, ok := tagSettings["COLUMN"]
			if !ok {
				columnName = namingStrategy.ColumnName("", field.Name)
			}

			taggedColumns = append(taggedColumns, columnName)
		}
	}

	return taggedColumns
}
