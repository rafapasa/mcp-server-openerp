package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/rafapasa/mcp-server-openerp/migrations"
)

func main() {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		// fallback para seu .env: usuario:senha@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "3306"
		}
		user := os.Getenv("DB_USER")
		dbpass := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if user == "" {
			log.Fatal("DATABASE_DSN ou DB_USER/DB_PASSWORD/DB_NAME não definidos")
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, dbpass, host, port, dbname)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("erro ao conectar no MySQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("erro ao pegar sql.DB: %v", err)
	}

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("mysql"); err != nil {
		log.Fatalf("dialect: %v", err)
	}

	dir := "."
	if len(os.Args) > 1 && os.Args[1] == "up" || len(os.Args) > 1 && os.Args[1] == "down" {
		// goose up / down
		cmd := os.Args[1]
		if cmd == "up" {
			if err := goose.Up(sqlDB, dir); err != nil {
				log.Fatalf("goose up failed: %v", err)
			}
			fmt.Println("✅ migrations up aplicadas")
		} else if cmd == "down" {
			if err := goose.Down(sqlDB, dir); err != nil {
				log.Fatalf("goose down failed: %v", err)
			}
			fmt.Println("✅ rollback aplicado")
		}
	} else {
		// default: up
		if err := goose.Up(sqlDB, dir); err != nil {
			log.Fatalf("goose up failed: %v", err)
		}
		fmt.Println("✅ migrations up aplicadas (default)")
	}
}
