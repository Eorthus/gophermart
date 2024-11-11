package main

import (
	"context"
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/Eorthus/gophermart/internal/accrual"
	"github.com/Eorthus/gophermart/internal/api"
	"github.com/Eorthus/gophermart/internal/config"
	"github.com/Eorthus/gophermart/internal/service"
	"github.com/Eorthus/gophermart/internal/storage"
)

func main() {
	// Инициализация логгера
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}
	logger.Info("Config loaded", zap.String("config", cfg.String()))

	// Инициализация хранилища
	ctx := context.Background()
	store, err := storage.NewDatabaseStorage(ctx, cfg.DatabaseURI)
	if err != nil {
		logger.Fatal("Failed to initialize storage", zap.Error(err))
	}
	defer store.Close()

	// Инициализация клиента системы начисления баллов
	accrualClient := accrual.NewClient(cfg.AccrualSystemAddress)

	//Создание обработчика заказов
	orderProcessor := service.NewOrderProcessor(store, accrualClient, logger)
	go orderProcessor.Start(ctx)

	// Инициализация сервисов
	userService := service.NewUserService(store)
	orderService := service.NewOrderService(store, accrualClient, *logger, orderProcessor)
	balanceService := service.NewBalanceService(store)

	// Инициализация маршрутизатора
	router := api.NewRouter(
		cfg,
		userService,
		orderService,
		balanceService,
		logger,
	)

	// Запуск сервера
	logger.Info("Starting server", zap.String("address", cfg.RunAddress))
	log.Fatal(http.ListenAndServe(cfg.RunAddress, router))
}
