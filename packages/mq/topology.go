package mq

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SetupTopology(url string) error {
	var conn *amqp.Connection
	var err error

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("⚠️ RabbitMQ 連線失敗 (嘗試 %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// 1. 宣告「死信交換機 (DLX)」
	// 當主佇列的訊息被 NACK 且不重入隊時，會被丟到這裡
	err = ch.ExchangeDeclare(
		"dlx_exchange",
		"direct",
		true, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	// 2. 宣告主要業務 Exchange (direct 類型)
	err = ch.ExchangeDeclare(
		"task_exchange",
		"direct",
		true, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	// 3. 宣告各個業務對應的「死信佇列 (DLQ)」
	// 這裡我們幫 stt 和 llm 分別建立對應的死信存放處
	businessQueues := []string{"stt-queue", "llm-queue"}

	for _, name := range businessQueues {
		dlqName := name + ".dlq"

		// A. 建立 DLQ
		_, err := ch.QueueDeclare(dlqName, true, false, false, false, nil)
		if err != nil {
			return err
		}

		// B. 將 DLQ 綁定到 DLX，Routing Key 就用原佇列名
		err = ch.QueueBind(dlqName, name, "dlx_exchange", false, nil)
		if err != nil {
			return err
		}

		// C. 建立「主業務佇列」，並指定它的 DLX
		args := amqp.Table{
			"x-dead-letter-exchange":    "dlx_exchange",
			"x-dead-letter-routing-key": name, // 失敗後丟往 DLX 時帶的 Key
		}

		_, err = ch.QueueDeclare(
			name,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			args,  // 這裡注入死信設定
		)
		if err != nil {
			return err
		}

		// D. 將業務佇列綁定到 Exchange，Routing Key 用佇列名
		err = ch.QueueBind(name, name, "task_exchange", false, nil)
		if err != nil {
			return err
		}

		log.Printf("✅ 已同步主佇列與死信配置: %s <-> %s", name, dlqName)
	}

	return nil
}
