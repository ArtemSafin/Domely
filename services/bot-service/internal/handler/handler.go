package handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ArtemSafin/Domely/services/bot-service/internal/client"
	tele "gopkg.in/telebot.v3"
)

// состояния диалога создания задачи
type dialogState int

const (
	stateIdle dialogState = iota
	stateAwaitTitle
	stateAwaitTaskType
	stateAwaitPriority
	stateAwaitReminder
	stateAwaitDueDate
	stateAwaitInterval
	stateAwaitAssignee
)

// сессия создания задачи для конкретного пользователя
type createSession struct {
	state    dialogState
	houseID  uuid.UUID
	userID   uuid.UUID
	req      client.CreateTaskRequest
	members  []client.User // для выбора assignee
}

type Handler struct {
	bot      *tele.Bot
	taskSvc  *client.TaskServiceClient
	sessions sync.Map // map[int64]*createSession
}

func New(bot *tele.Bot, taskSvc *client.TaskServiceClient) *Handler {
	return &Handler{bot: bot, taskSvc: taskSvc}
}

func (h *Handler) Register() {
	h.bot.Handle("/start", h.handleStart)
	h.bot.Handle("/help", h.handleHelp)
	h.bot.Handle("/house", h.handleHouse)
	h.bot.Handle("/add", h.handleAdd)
	h.bot.Handle("/list", h.handleList)
	h.bot.Handle("/done", h.handleDone)
	h.bot.Handle("/delete", h.handleDelete)
	h.bot.Handle("/history", h.handleHistory)

	// все текстовые сообщения вне команд идут в диалог
	h.bot.Handle(tele.OnText, h.handleText)

	// inline кнопки
	h.bot.Handle(tele.OnCallback, h.handleCallback)
}

// --- /start ---

func (h *Handler) handleStart(c tele.Context) error {
	ctx := context.Background()
	telegramID := c.Sender().ID
	name := strings.TrimSpace(c.Sender().FirstName + " " + c.Sender().LastName)

	// регистрируем пользователя (idempotent)
	user, err := h.taskSvc.RegisterUser(ctx, telegramID, name)
	if err != nil {
		log.Printf("register user %d: %v", telegramID, err)
		return c.Send("Ошибка при регистрации. Попробуй ещё раз.")
	}

	// проверяем есть ли у пользователя дома
	houses, err := h.taskSvc.GetHousesByUser(ctx, user.ID)
	if err != nil {
		return c.Send("Ошибка при загрузке домов.")
	}

	if len(houses) == 0 {
		return c.Send(
			fmt.Sprintf("Привет, %s! 👋\n\nДобро пожаловать в *Homie*.\n\nУ тебя пока нет ни одного дома. Создай его командой:\n`/house new Мой дом`", user.Name),
			tele.ModeMarkdown,
		)
	}

	return c.Send(
		fmt.Sprintf("Привет, %s! 👋\n\nИспользуй /help чтобы увидеть список команд.", user.Name),
		tele.ModeMarkdown,
	)
}

// --- /help ---

func (h *Handler) handleHelp(c tele.Context) error {
	text := `*Homie* — напоминания для дома 🏠

*Дом*
/house new <название> — создать дом
/house list — список домов
/house invite <telegram\_id> — пригласить члена семьи

*Задачи*
/add — создать задачу
/list — список активных задач
/done <id> — отметить выполненной
/delete <id> — удалить задачу
/history — история выполнений`

	return c.Send(text, tele.ModeMarkdown)
}

// --- /house ---

func (h *Handler) handleHouse(c tele.Context) error {
	ctx := context.Background()
	args := c.Args()

	if len(args) == 0 {
		return c.Send("Используй:\n`/house new <название>`\n`/house list`\n`/house invite <telegram_id>`", tele.ModeMarkdown)
	}

	user, err := h.getOrRegisterUser(ctx, c.Sender())
	if err != nil {
		return c.Send("Ошибка авторизации.")
	}

	switch args[0] {
	case "new":
		if len(args) < 2 {
			return c.Send("Укажи название: `/house new Мой дом`", tele.ModeMarkdown)
		}
		name := strings.Join(args[1:], " ")
		house, err := h.taskSvc.CreateHouse(ctx, name, user.ID)
		if err != nil {
			return c.Send("Ошибка при создании дома.")
		}
		return c.Send(fmt.Sprintf("🏠 Дом *%s* создан!", house.Name), tele.ModeMarkdown)

	case "list":
		houses, err := h.taskSvc.GetHousesByUser(ctx, user.ID)
		if err != nil {
			return c.Send("Ошибка при загрузке домов.")
		}
		if len(houses) == 0 {
			return c.Send("У тебя пока нет домов. Создай: `/house new Название`", tele.ModeMarkdown)
		}
		var sb strings.Builder
		sb.WriteString("*Твои дома:*\n\n")
		for _, h := range houses {
			sb.WriteString(fmt.Sprintf("🏠 %s (`%s`)\n", h.Name, h.ID))
		}
		return c.Send(sb.String(), tele.ModeMarkdown)

	case "invite":
		if len(args) < 2 {
			return c.Send("Укажи telegram_id: `/house invite 123456789`", tele.ModeMarkdown)
		}
		inviteeID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return c.Send("Неверный telegram_id.")
		}
		invitee, err := h.taskSvc.GetUserByTelegramID(ctx, inviteeID)
		if err != nil {
			return c.Send("Пользователь не найден. Он должен сначала написать боту /start.")
		}
		houses, err := h.taskSvc.GetHousesByUser(ctx, user.ID)
		if err != nil || len(houses) == 0 {
			return c.Send("У тебя нет домов.")
		}
		// приглашаем в первый дом (TODO: выбор дома если несколько)
		if err := h.taskSvc.InviteMember(ctx, houses[0].ID, invitee.ID); err != nil {
			return c.Send("Ошибка при добавлении участника.")
		}
		return c.Send(fmt.Sprintf("✅ %s добавлен в дом *%s*!", invitee.Name, houses[0].Name), tele.ModeMarkdown)
	}

	return c.Send("Неизвестная команда. Используй /help.")
}

// --- /add — запускает диалог ---

func (h *Handler) handleAdd(c tele.Context) error {
	ctx := context.Background()

	user, err := h.getOrRegisterUser(ctx, c.Sender())
	if err != nil {
		return c.Send("Ошибка авторизации.")
	}

	houses, err := h.taskSvc.GetHousesByUser(ctx, user.ID)
	if err != nil || len(houses) == 0 {
		return c.Send("Сначала создай дом: `/house new Название`", tele.ModeMarkdown)
	}

	// сохраняем сессию
	session := &createSession{
		state:   stateAwaitTitle,
		houseID: houses[0].ID,
		userID:  user.ID,
		req: client.CreateTaskRequest{
			HouseID:   houses[0].ID,
			CreatedBy: user.ID,
		},
	}
	h.sessions.Store(c.Sender().ID, session)

	return c.Send("📝 Введи название задачи:")
}

// --- обработка текста в диалоге ---

func (h *Handler) handleText(c tele.Context) error {
	val, ok := h.sessions.Load(c.Sender().ID)
	if !ok {
		return nil
	}

	session := val.(*createSession)
	text := strings.TrimSpace(c.Text())

	switch session.state {
	case stateAwaitTitle:
		session.req.Title = text
		session.state = stateAwaitTaskType
		return c.Send("Тип задачи?", taskTypeKeyboard())

	case stateAwaitDueDate:
		t, err := parseDateTime(text)
		if err != nil {
			return c.Send("Не понял дату. Попробуй формат: `25.01.2025 15:00`", tele.ModeMarkdown)
		}
		session.req.DueAt = &t
		session.state = stateAwaitAssignee

		members, err := h.taskSvc.GetHouseMembers(context.Background(), session.houseID)
		if err != nil {
			return c.Send("Ошибка при загрузке членов дома.")
		}
		session.members = members
		return c.Send("Кому назначить?", assigneeKeyboard(members))

	case stateAwaitInterval:
		days, err := strconv.Atoi(text)
		if err != nil || days <= 0 {
			return c.Send("Введи количество дней числом, например: `30`", tele.ModeMarkdown)
		}
		session.req.IntervalDays = &days
		session.state = stateAwaitAssignee

		members, err := h.taskSvc.GetHouseMembers(context.Background(), session.houseID)
		if err != nil {
			return c.Send("Ошибка при загрузке членов дома.")
		}
		session.members = members
		return c.Send("Кому назначить?", assigneeKeyboard(members))
	}

	return nil
}

// --- inline кнопки ---

func (h *Handler) handleCallback(c tele.Context) error {
	val, ok := h.sessions.Load(c.Sender().ID)
	if !ok {
		return c.Respond()
	}

	session := val.(*createSession)
	data := c.Callback().Data

	switch session.state {
	case stateAwaitTaskType:
		session.req.TaskType = data
		session.state = stateAwaitPriority
		c.Respond()
		return c.Send("Приоритет?", priorityKeyboard())

	case stateAwaitPriority:
		session.req.Priority = data
		session.state = stateAwaitReminder
		c.Respond()
		return c.Send("Тип напоминания?", reminderKeyboard())

	case stateAwaitReminder:
		session.req.ReminderStrategy = data
		c.Respond()

		if session.req.TaskType == "recurring" {
			session.state = stateAwaitInterval
			return c.Send("Каждые сколько дней повторять? Введи число:")
		}
		session.state = stateAwaitDueDate
		return c.Send("Дата и время? Формат: `25.01.2025 15:00`", tele.ModeMarkdown)

	case stateAwaitAssignee:
		c.Respond()
		if data != "all" {
			id, err := uuid.Parse(data)
			if err == nil {
				session.req.AssignedTo = &id
			}
		}
		return h.finishCreateTask(c, session)
	}

	return c.Respond()
}

func (h *Handler) finishCreateTask(c tele.Context, session *createSession) error {
	h.sessions.Delete(c.Sender().ID)

	task, err := h.taskSvc.CreateTask(context.Background(), session.req)
	if err != nil {
		log.Printf("create task: %v", err)
		return c.Send("Ошибка при создании задачи. Попробуй ещё раз.")
	}

	return c.Send(fmt.Sprintf("✅ Задача создана!\n\n*%s*\nПриоритет: %s | Тип: %s",
		task.Title, priorityLabel(task.Priority), taskTypeLabel(task.TaskType),
	), tele.ModeMarkdown)
}

// --- /list ---

func (h *Handler) handleList(c tele.Context) error {
	ctx := context.Background()

	user, err := h.getOrRegisterUser(ctx, c.Sender())
	if err != nil {
		return c.Send("Ошибка авторизации.")
	}

	houses, err := h.taskSvc.GetHousesByUser(ctx, user.ID)
	if err != nil || len(houses) == 0 {
		return c.Send("У тебя нет домов.")
	}

	tasks, err := h.taskSvc.GetTasksByHouse(ctx, houses[0].ID)
	if err != nil {
		return c.Send("Ошибка при загрузке задач.")
	}

	if len(tasks) == 0 {
		return c.Send("Задач нет. Создай первую: /add")
	}

	var sb strings.Builder
	sb.WriteString("*Активные задачи:*\n\n")
	for i, t := range tasks {
		sb.WriteString(fmt.Sprintf("%d. %s %s\n",
			i+1, priorityIcon(t.Priority), t.Title))
		if t.DueAt != nil {
			sb.WriteString(fmt.Sprintf("   📅 %s\n", t.DueAt.Format("02.01.2006 15:04")))
		}
		sb.WriteString(fmt.Sprintf("   `id: %s`\n\n", t.ID))
	}

	return c.Send(sb.String(), tele.ModeMarkdown)
}

// --- /done ---

func (h *Handler) handleDone(c tele.Context) error {
	ctx := context.Background()
	args := c.Args()

	if len(args) == 0 {
		return c.Send("Укажи id задачи: `/done <id>`", tele.ModeMarkdown)
	}

	taskID, err := uuid.Parse(args[0])
	if err != nil {
		return c.Send("Неверный id задачи.")
	}

	user, err := h.getOrRegisterUser(ctx, c.Sender())
	if err != nil {
		return c.Send("Ошибка авторизации.")
	}

	if err := h.taskSvc.CompleteTask(ctx, taskID, user.ID); err != nil {
		return c.Send("Ошибка при выполнении задачи.")
	}

	return c.Send("✅ Задача выполнена!")
}

// --- /delete ---

func (h *Handler) handleDelete(c tele.Context) error {
	args := c.Args()

	if len(args) == 0 {
		return c.Send("Укажи id задачи: `/delete <id>`", tele.ModeMarkdown)
	}

	taskID, err := uuid.Parse(args[0])
	if err != nil {
		return c.Send("Неверный id задачи.")
	}

	// кнопка подтверждения
	kb := &tele.ReplyMarkup{}
	btnYes := kb.Data("🗑 Удалить", "confirm_delete", taskID.String())
	btnNo := kb.Data("Отмена", "cancel_delete")
	kb.Inline(kb.Row(btnYes, btnNo))

	return c.Send("Удалить задачу?", kb)
}

// --- /history ---

func (h *Handler) handleHistory(c tele.Context) error {
	ctx := context.Background()

	user, err := h.getOrRegisterUser(ctx, c.Sender())
	if err != nil {
		return c.Send("Ошибка авторизации.")
	}

	houses, err := h.taskSvc.GetHousesByUser(ctx, user.ID)
	if err != nil || len(houses) == 0 {
		return c.Send("У тебя нет домов.")
	}

	// используем GetTasksByHouse и фильтруем неактивные как историю
	// в будущем можно добавить отдельный endpoint
	return c.Send("📋 История пока в разработке. Используй /list для активных задач.")
}

// --- helpers ---

func (h *Handler) getOrRegisterUser(ctx context.Context, sender *tele.User) (*client.User, error) {
	user, err := h.taskSvc.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		name := strings.TrimSpace(sender.FirstName + " " + sender.LastName)
		return h.taskSvc.RegisterUser(ctx, sender.ID, name)
	}
	return user, nil
}

func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		"02.01.2006 15:04",
		"2.01.2006 15:04",
		"02.01.2006",
		"2006-01-02 15:04",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown format: %s", s)
}

// --- клавиатуры ---

func taskTypeKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(
		kb.Data("🔂 Повторяющаяся", "recurring"),
		kb.Data("1️⃣ Разовая", "one_time"),
	))
	return kb
}

func priorityKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(
		kb.Data("🟢 Низкий", "low"),
		kb.Data("🟡 Обычный", "normal"),
		kb.Data("🔴 Высокий", "high"),
	))
	return kb
}

func reminderKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(
		kb.Data("🔔 Простое", "simple"),
		kb.Data("📅 За день", "advance"),
		kb.Data("🤝 Встреча", "meeting"),
	))
	return kb
}

func assigneeKeyboard(members []client.User) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	rows = append(rows, kb.Row(kb.Data("👥 Всем", "all")))
	for _, m := range members {
		rows = append(rows, kb.Row(kb.Data("👤 "+m.Name, m.ID.String())))
	}
	kb.Inline(rows...)
	return kb
}

func priorityIcon(p string) string {
	switch p {
	case "high":
		return "🔴"
	case "low":
		return "🟢"
	default:
		return "🟡"
	}
}

func priorityLabel(p string) string {
	switch p {
	case "high":
		return "Высокий"
	case "low":
		return "Низкий"
	default:
		return "Обычный"
	}
}

func taskTypeLabel(t string) string {
	if t == "recurring" {
		return "Повторяющаяся"
	}
	return "Разовая"
}
