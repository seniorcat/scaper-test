package cmd

import (
	"log"
	"time"

	"github.com/seniorcat/scraper-test/config" // Пакет конфигурации
	"github.com/seniorcat/scraper-test/worker"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Параметры командной строки
var (
	parseType   int
	maxRecipes  int
	concurrency int

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Запуск парсера",
		Long:  "Запуск парсера с параметрами: тип парсинга, количество рецептов и одновременных потоков.",
		Run:   runParser,
	}
)

// init добавляет флаги для команды run
func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntVarP(&parseType, "type", "t", 1, "Тип парсинга: 1 - полный, 2 - только категории")
	runCmd.Flags().IntVarP(&concurrency, "concurrency", "g", 5, "Количество одновременных потоков")
}

// runParser запускает парсер с заданными параметрами
func runParser(cmd *cobra.Command, args []string) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Не удалось инициализировать логгер: %v", err)
	}
	defer logger.Sync()

	// Инициализация конфигурации
	cfgWrapper, err := config.New("config.yaml")
	if err != nil {
		logger.Fatal("Ошибка загрузки конфигурации", zap.Error(err))
	}

	// Считывание параметров из конфигурации
	timeout := cfgWrapper.GetInt64("worker.timeout")
	maxRecipes := cfgWrapper.GetInt64("worker.maxRecipes")

	// Создание воркеров с использованием конфигурации
	categoryWorker := worker.NewCategoryWorker(logger, time.Duration(timeout)*time.Second)
	recipeWorker := worker.NewRecipeWorker(logger, int(maxRecipes), time.Duration(timeout)*time.Second)

	switch parseType {
	case 1:
		log.Println("Запуск полного парсинга...")

		categories, err := categoryWorker.Start()
		if err != nil {
			logger.Error("Ошибка при парсинге категорий", zap.Error(err))
			return
		}

		for _, category := range categories {
			logger.Info("Категория", zap.String("Name", category.Name), zap.String("Href", category.Href))

			recipes, err := recipeWorker.Start(category)
			if err != nil {
				logger.Error("Ошибка при парсинге рецептов", zap.String("Category", category.Name), zap.Error(err))
				continue
			}

			for _, recipe := range recipes {
				logger.Info("Рецепт", zap.String("Name", recipe.Name), zap.String("Href", recipe.Href))
			}
		}

	case 2:
		log.Println("Запуск парсинга категорий...")
		categories, err := categoryWorker.Start()
		if err != nil {
			logger.Error("Ошибка при парсинге категорий", zap.Error(err))
			return
		}

		for _, category := range categories {
			logger.Info("Категория", zap.String("Name", category.Name), zap.String("Href", category.Href))
		}

	default:
		log.Printf("Неизвестный тип парсинга: %d\n", parseType)
	}
}
