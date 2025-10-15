package storage

const (
	createUserTable = `CREATE TABLE IF NOT EXISTS
					    users(
							login    TEXT PRIMARY KEY,
							password TEXT
						)`
	createOrdersTable = `CREATE TABLE IF NOT EXISTS
						 orders(
							number TEXT PRIMARY KEY,
							login TEXT,
							time   TEXT,
							status TEXT,
							accrual INT
						 )`
	createHistoryTable = `CREATE TABLE IF NOT EXISTS
							ordersHistory(
								number      TEXT,
								withdraw    INT,
								time        TEXT
							)`

	insertUser = `INSERT INTO users(login, password) VALUES($1, $2)`
	selectUser = `SELECT password FROM users WHERE login = $1`

	selectOrder      = `SELECT login FROM orders WHERE number = $1`
	selectUserOrders = `SELECT number, login, time, status, accrual FROM orders WHERE login = $1 AND time IS NOT NULL`
	insertOrder      = `INSERT INTO orders(number, login, time, status, accrual) VALUES($1, $2, $3, 'NEW', 0)`

	selectProcessingOrders = `SELECT number FROM orders WHERE status != $1 AND status != $2 AND time IS NOT NULL`
	updateOrdersStatus     = `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`

	addOrderHistory   = `INSERT INTO ordersHistory(number, withdraw, time) VALUES($1, $2, $3)`
	selectUserHistory = `SELECT h.withdraw 
						 FROM ordersHistory as h 
						 JOIN orders AS o ON o.number = h.number 
						 WHERE o.login = $1`
	selectWithdrawHistory = `SELECT h.number, h.withdraw, h.time
						 FROM ordersHistory as h 
						 JOIN orders AS o ON h.number = o.number
						 WHERE o.login = $1`
)
