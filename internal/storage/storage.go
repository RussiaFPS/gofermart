package storage

import (
	"context"
	"errors"
	"github.com/RussiaFPS/gofermart/internal/config"
	"github.com/RussiaFPS/gofermart/internal/model"
	"sort"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

type Storages interface {
	AddUser(ctx context.Context, user model.User) error
	AuthUser(ctx context.Context, user model.User) (string, error)
	AddOrder(ctx context.Context, number string, login string) error
	GetOrders(ctx context.Context, login string) ([]model.OrdersResponse, error)
	WriteWithdraw(ctx context.Context, withdraw model.OrderWithdraw, login string) error
	GetBalance(ctx context.Context, login string) (model.Balance, error)
	GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error)
	GetOrdersForUpdate(ctx context.Context) ([]string, error)
	UpdateOrders(ctx context.Context, accrualSysResponse []model.PointsAppResponse)
}

type DBStruct struct {
	pgxPool *pgxpool.Pool
	log     *logrus.Logger
	cfg     *config.Config
}

func NewStorage(ctx context.Context, cfg *config.Config, log *logrus.Logger) (*DBStruct, error) {
	pgxPool, err := InitConnection(ctx, cfg.Database, log)
	if err != nil {
		return nil, err
	}
	return &DBStruct{
		pgxPool: pgxPool,
		log:     log,
		cfg:     cfg,
	}, nil
}

func InitConnection(ctx context.Context, connString string, log *logrus.Logger) (*pgxpool.Pool, error) {
	pgxPool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		log.Fatal(err.Error())
		return nil, err
	}

	if _, err = pgxPool.Exec(ctx, createUserTable); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	if _, err = pgxPool.Exec(ctx, createOrdersTable); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	if _, err = pgxPool.Exec(ctx, createHistoryTable); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return pgxPool, nil
}

func (db *DBStruct) Close() {
	db.pgxPool.Close()
}

func (db *DBStruct) AddUser(ctx context.Context, user model.User) error {
	var pgxError *pgconn.PgError
	_, err := db.pgxPool.Exec(ctx, insertUser, user.Login, user.Password)
	if errors.As(err, &pgxError) {
		if pgxError.Code == pgerrcode.UniqueViolation {
			db.log.Error(model.ErrLoginExists.Error())
			return model.ErrLoginExists
		}
	}
	if err != nil {
		db.log.Error(err.Error())
	}
	return err
}

func (db *DBStruct) AuthUser(ctx context.Context, user model.User) (string, error) {
	var checkUser model.User
	row := db.pgxPool.QueryRow(ctx, selectUser, user.Login)
	err := row.Scan(&checkUser.Password)
	if err != nil {
		db.log.Error(err.Error())
		return "", err
	}

	return checkUser.Password, err
}

func (db *DBStruct) AddOrder(ctx context.Context, number string, login string) error {
	var user string

	t := time.Now().Format(time.RFC3339)

	row := db.pgxPool.QueryRow(ctx, selectOrder, number)
	err := row.Scan(&user)

	if err == nil {
		if user == login {
			db.log.Error(model.ErrOrderExistsSameUser.Error())
			return model.ErrOrderExistsSameUser
		}
		db.log.WithFields(logrus.Fields{
			"user": user}).Error(model.ErrOrderExistsDiffUser.Error())
		return model.ErrOrderExistsDiffUser
	}
	db.log.WithFields(logrus.Fields{
		"number": number,
		"login":  login,
		"t":      t}).Info("Запись заказа в таблицу orderTable")

	_, err = db.pgxPool.Exec(ctx, insertOrder, number, login, t)
	if err != nil {
		db.log.Error(err.Error())
	}
	return err
}

func (db *DBStruct) GetOrders(ctx context.Context, login string) ([]model.OrdersResponse, error) {
	var timeStr string
	var accrualInt int32
	var orderResp model.OrdersResponse
	var orders []model.OrdersResponse

	db.log.WithFields(
		logrus.Fields{
			"login": login,
		}).Info("Выбираем заказы для пользователя")
	rows, err := db.pgxPool.Query(ctx, selectUserOrders, login)
	if err != nil {
		db.log.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&orderResp.Number, &orderResp.Login, &timeStr, &orderResp.Status, &accrualInt)
		if err != nil {
			db.log.Error(err.Error())
			return nil, err
		}
		orderResp.Time, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			db.log.Error(err.Error())
		}
		orderResp.Accrual = float64(accrualInt)
		orderResp.Accrual = orderResp.Accrual / 100
		db.log.WithFields(logrus.Fields{
			"time":    timeStr,
			"number":  orderResp.Number,
			"status":  orderResp.Status,
			"accrual": orderResp.Accrual,
		}).Info("Получили строку с заказом")
		orders = append(orders, orderResp)
	}

	if rows.Err() != nil {
		db.log.Error(rows.Err().Error())
		return nil, err
	}

	if orders == nil {
		db.log.Info("в orders пусто")
		return nil, errors.New("в orders пусто")
	}
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].Time.Before(orders[j].Time)
	})
	return orders, nil
}

func (db *DBStruct) WriteWithdraw(ctx context.Context, withdraw model.OrderWithdraw, login string) error {
	tx, err := db.pgxPool.Begin(ctx)
	if err != nil {
		db.log.Error("Не удалось начать транзакцию: ", err.Error())
		return err
	}
	defer tx.Rollback(ctx)
	db.log.WithFields(logrus.Fields{
		"number":   withdraw.Number,
		"withdraw": withdraw.Withdraw,
	}).Info("Запись в таблицу OrdersHistory")

	_, err = tx.Exec(ctx, addOrderHistory, withdraw.Number, withdraw.Withdraw, time.Now().Format(time.RFC3339))
	if err != nil {
		db.log.Error("Не удалось добавить в OrdersHistory: ", err.Error())
		return err
	}

	db.log.WithFields(logrus.Fields{
		"number": withdraw.Number,
		"login":  login,
	}).Info("Запись в таблицу orders")

	_, err = tx.Exec(ctx, insertOrder, withdraw.Number, login, nil)
	if err != nil {
		db.log.Error("Не удалось добавить заказ: ", err.Error())
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		db.log.Error("Не удалось зафиксировать транзакцию: ", err.Error())
		return err
	}

	db.log.Info("Транзакция по выводу средств успешно завершена")
	return nil
}

func (db *DBStruct) GetOrdersForUpdate(ctx context.Context) ([]string, error) {
	var orderNumber string
	var orderNumbers []string

	rows, err := db.pgxPool.Query(ctx, selectProcessingOrders, "INVALID", "PROCESSED")
	if err != nil {
		db.log.Error(err.Error())
		return nil, err
	}

	for rows.Next() {
		err := rows.Scan(&orderNumber)
		if err != nil {
			db.log.Error(err.Error())
			continue
		}
		db.log.WithFields(logrus.Fields{"orderNumber": orderNumber}).Info("Выбран заказ для запроса статуса")
		orderNumbers = append(orderNumbers, orderNumber)
	}

	if rows.Err() != nil {
		db.log.Error(rows.Err().Error())
		return nil, rows.Err()
	}
	return orderNumbers, nil
}

func (db *DBStruct) UpdateOrders(ctx context.Context, accrualSysResponse []model.PointsAppResponse) {
	var accrual int32

	batch := &pgx.Batch{}
	for _, response := range accrualSysResponse {
		accrual = int32(response.Accrual * 100)
		batch.Queue(updateOrdersStatus, response.Status, accrual, response.Number)
		batch.Queue(addOrderHistory, response.Number, accrual, time.Now().Format(time.RFC3339))
		db.log.WithFields(logrus.Fields{
			"number":  response.Number,
			"status":  response.Status,
			"accrual": accrual,
		}).Info("Обновление заказа")
	}
	batchReq := db.pgxPool.SendBatch(ctx, batch)
	_, err := batchReq.Exec()
	if err != nil {
		db.log.Error(err.Error())
	}
	batchReq.Close()
}

func (db *DBStruct) GetBalance(ctx context.Context, login string) (model.Balance, error) {
	var balance model.Balance
	var withdrawFloat float64
	var withdraw int32

	rows, err := db.pgxPool.Query(ctx, selectUserHistory, login)
	if err != nil {
		db.log.Error(rows.Err().Error())
		return model.Balance{}, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&withdraw)
		if err != nil {
			db.log.Error(err.Error())
			return model.Balance{}, err
		}
		withdrawFloat = float64(withdraw)
		db.log.WithFields(logrus.Fields{"withdraw": withdraw}).Info("Баланс")
		balance.Balance += withdrawFloat
		if withdrawFloat < 0 {
			balance.Withdrawn += withdrawFloat
		}
	}

	if rows.Err() != nil {
		db.log.Error(rows.Err().Error())
		return model.Balance{}, rows.Err()
	}

	return balance, nil
}

func (db *DBStruct) GetWithdrawals(ctx context.Context, login string) ([]model.OrderWithdraw, error) {
	var userWithdraw model.OrderWithdraw
	var allWithdrawals []model.OrderWithdraw
	var withdraw int32
	var timeUpl string

	rows, err := db.pgxPool.Query(ctx, selectWithdrawHistory, login)
	if err != nil {
		db.log.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&userWithdraw.Number, &withdraw, &timeUpl)
		if err != nil {
			db.log.Error(err.Error())
			return nil, err
		}
		if withdraw > 0 {
			continue
		}
		userWithdraw.Time, err = time.Parse(time.RFC3339, timeUpl)
		if err != nil {
			db.log.Error(err.Error())
		}
		userWithdraw.Withdraw = float64(withdraw)
		userWithdraw.Withdraw = -userWithdraw.Withdraw / 100

		db.log.WithFields(logrus.Fields{
			"number":   userWithdraw.Number,
			"withdraw": userWithdraw.Withdraw,
			"time":     userWithdraw.Time,
		}).Info("Списание")
		allWithdrawals = append(allWithdrawals, userWithdraw)
	}
	if err := rows.Err(); err != nil {
		db.log.Error(err.Error())
		return nil, err
	}
	if allWithdrawals == nil {
		db.log.Error(model.ErrNoWithdrawals.Error())
		return nil, model.ErrNoWithdrawals
	}

	sort.SliceStable(allWithdrawals, func(i, j int) bool {
		return allWithdrawals[i].Time.Before(allWithdrawals[j].Time)
	})
	return allWithdrawals, nil
}
