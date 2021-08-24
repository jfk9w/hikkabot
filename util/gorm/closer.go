package gorm

import "gorm.io/gorm"

type Closer gorm.DB

func (c *Closer) Unmask() *gorm.DB {
	return (*gorm.DB)(c)
}

func (c *Closer) Close() error {
	db, err := c.Unmask().DB()
	if err != nil {
		return err
	}

	return db.Close()
}
