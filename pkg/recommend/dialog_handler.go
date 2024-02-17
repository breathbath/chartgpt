package recommend

import (
	"breathbathChatGPT/pkg/utils"
	"context"
	logging "github.com/sirupsen/logrus"
	"gopkg.in/errgo.v2/errors"
	"gorm.io/gorm"
	"strings"
	"time"
)

const (
	ConversationHistoryWindow = -time.Minute * 30
	ActionTypeFilterPrompt    = "filter_prompt"
	FiltersContext            = "filters"
	ActionTypeRecommendation  = "recommendation"
	PathSeparator             = "->"
	RandPath                  = "rand"
	RecommendationPath        = ActionTypeRecommendation
	PromptSecondaryPath       = "promptSecondary"
	PromptRandomSecondaryPath = "promptRandomSecondary"
	PromptPrimaryPath         = "promptPrimary"
)

type Action struct {
	Type    string                 `gorm:"type:string"`
	Context map[string]interface{} `gorm:"type:string;serializer:json"`
}

func (a Action) GetFilters() []string {
	filtersI, ok := a.Context[FiltersContext]
	if !ok {
		return []string{}
	}

	filters, ok := filtersI.([]string)
	if !ok {
		return []string{}
	}

	return filters
}

func (a Action) IsRecommendation() bool {
	return a.Type == ActionTypeRecommendation
}

type DialogItem struct {
	gorm.Model
	InputFilter  *WineFilter `gorm:"type:string;serializer:json"`
	OutputAction *Action     `gorm:"embedded;embeddedPrefix:output_action_"`
	UserID       string
	Path         string
}

func (di *DialogItem) AddToPath(input string) *DialogItem {
	if di.Path == "" {
		di.Path = input
		return di
	}

	di.Path += PathSeparator + input

	return di
}

func (di *DialogItem) MatchesPath(input string) bool {
	if di == nil {
		return false
	}
	return strings.Contains(di.Path, input)
}

func (di *DialogItem) MatchesLastPath(input string) bool {
	if di == nil {
		return false
	}
	return strings.HasSuffix(di.Path, input)
}

func (di *DialogItem) AddToPathRecommend() *DialogItem {
	return di.AddToPath(RecommendationPath)
}

func (di *DialogItem) AddToPathRandom() *DialogItem {
	return di.AddToPath(RandPath)
}

type Dialog []DialogItem

func (d Dialog) Previous() *DialogItem {
	if len(d) == 0 {
		return nil
	}

	return &d[0]
}

func (d Dialog) HasPreviousFilterPrompt() bool {
	previous := d.Previous()
	return previous != nil &&
		previous.OutputAction != nil &&
		previous.OutputAction.Type == ActionTypeFilterPrompt
}

type DialogHandler struct {
	db *gorm.DB
}

func NewDialogHandler(db *gorm.DB) *DialogHandler {
	return &DialogHandler{
		db: db,
	}
}

func (dh DialogHandler) DecideAction(
	ctx context.Context,
	f *WineFilter,
	userID string,
) (*Action, error) {
	log := logging.WithContext(ctx)

	log.Debugf("going to decide about next dialog action for filters: %s", f.String())

	dialog, err := dh.loadDialog(userID)
	if err != nil {
		return nil, err
	}

	currentDialogItem := &DialogItem{
		UserID:      userID,
		InputFilter: f,
	}
	defer func() {
		if currentDialogItem.OutputAction != nil && currentDialogItem.OutputAction.Type != "" {
			dh.db.Create(currentDialogItem)
		}
	}()

	if f.Name != "" {
		return dh.handleNameFilter(ctx, currentDialogItem)
	}

	if dh.hasNotEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert(f, dialog) {
		return dh.handleHasNotEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert(ctx, currentDialogItem, dialog)
	}

	if dh.allPrimarySomeSecondaryAndSomeExpertFilters(f, dialog) {
		return dh.handleAllPrimarySomeSecondaryAndSomeExpertFilters(ctx, currentDialogItem)
	}

	if dh.somePrimaryNoSecondarySomeExpertFilters(f, dialog) {
		return dh.handleSomePrimaryNoSecondarySomeExpertFilters(ctx, currentDialogItem, dialog)
	}

	if dh.somePrimaryNoExpertFilters(f, dialog) {
		return dh.handleSomePrimaryNoExpertFilters(ctx, currentDialogItem, dialog)
	}

	if dh.hasAllPrimaryFiltersNoSecondaryFilters(f, dialog) {
		return dh.handleAllPrimaryFiltersNoSecondaryFilters(ctx, currentDialogItem, dialog)
	}

	log.Debug("falling back to recommendation as no decision branch was entered")
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) hasAllPrimaryFiltersNoSecondaryFilters(f *WineFilter, dialog *Dialog) bool {
	if dialog.HasPreviousFilterPrompt() {
		return dialog.Previous().MatchesPath("allPrimaryFiltersNoSecondaryFilters")
	}
	hasAllPrimaryFilters := f.GetPrimaryFiltersCount() >= f.GetTotalPrimaryFiltersCount()

	return hasAllPrimaryFilters && (!f.HasSecondaryFilters() && f.HasExpertFilters() || f.HasSecondaryFilters() && !f.HasExpertFilters())
}

func (dh DialogHandler) somePrimaryNoExpertFilters(f *WineFilter, dialog *Dialog) bool {
	if dialog.HasPreviousFilterPrompt() {
		return dialog.Previous().MatchesPath("somePrimaryNoExpertFilters")
	}

	primaryFiltersCount := f.GetPrimaryFiltersCount()

	return primaryFiltersCount < f.GetTotalPrimaryFiltersCount() && !f.HasExpertFilters()
}

func (dh DialogHandler) somePrimaryNoSecondarySomeExpertFilters(f *WineFilter, dialog *Dialog) bool {
	if dialog.HasPreviousFilterPrompt() {
		return dialog.Previous().MatchesPath("somePrimaryNoSecondarySomeExpertFilters")
	}

	primaryFiltersCount := f.GetPrimaryFiltersCount()

	return primaryFiltersCount < f.GetTotalPrimaryFiltersCount() &&
		primaryFiltersCount > 0 &&
		!f.HasSecondaryFilters() &&
		f.HasExpertFilters()
}

func (dh DialogHandler) hasNotEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert(f *WineFilter, dialog *Dialog) bool {
	if dialog.HasPreviousFilterPrompt() {
		return dialog.Previous().MatchesPath("notEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert")
	}

	primaryFiltersCount := f.GetPrimaryFiltersCount()
	return primaryFiltersCount < f.GetTotalPrimaryFiltersCount() &&
		f.HasSecondaryFilters() &&
		f.HasExpertFilters()
}

func (dh DialogHandler) allPrimarySomeSecondaryAndSomeExpertFilters(f *WineFilter, dialog *Dialog) bool {
	if dialog.HasPreviousFilterPrompt() {
		return dialog.Previous().MatchesPath("allPrimarySomeSecondaryAndSomeExpertFilters")
	}

	primaryFiltersCount := f.GetPrimaryFiltersCount()

	return primaryFiltersCount == f.GetTotalPrimaryFiltersCount() && f.HasSecondaryFilters() && f.HasExpertFilters()
}

func (dh DialogHandler) handleSomePrimaryNoExpertFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog *Dialog,
) (*Action, error) {
	currentDialogItem.AddToPath("somePrimaryNoExpertFilters")
	if dh.almostAllPrimaryNoExpertFilters(currentDialogItem.InputFilter, dialog, currentDialogItem) {
		return dh.handleAlmostAllPrimaryNoExpertFilters(ctx, currentDialogItem, dialog)
	}

	return dh.handleTooFewPrimaryNoExpertFilters(ctx, currentDialogItem, dialog)
}

func (dh DialogHandler) almostAllPrimaryNoExpertFilters(f *WineFilter, dialog *Dialog, currentDialogItem *DialogItem) bool {
	if dialog.HasPreviousFilterPrompt() {
		return dialog.Previous().MatchesPath("almostAllPrimaryNoExpertFilters")
	}

	primaryFiltersCount := f.GetPrimaryFiltersCount()

	return primaryFiltersCount == currentDialogItem.InputFilter.GetTotalPrimaryFiltersCount()-1
}

func (dh DialogHandler) handleAlmostAllPrimaryNoExpertFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog *Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has almost all primary and no expert filters")

	currentDialogItem.AddToPath("almostAllPrimaryNoExpertFilters")

	if !dialog.Previous().MatchesLastPath(PromptSecondaryPath) {
		randomSecondaryFilters := currentDialogItem.InputFilter.GetRandomSecondaryFilters()
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: randomSecondaryFilters,
			},
		}
		log.Debugf("did not detect previous prompts, will prompt random secondary filters %+v", randomSecondaryFilters)
		currentDialogItem.AddToPath(PromptSecondaryPath)
		return currentDialogItem.OutputAction, nil
	}

	currentDialogItem.AddToPathRecommend()
	log.Debug("detected previous prompts, decided randomly not to prompt secondary filters again and give a recommendation")
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) handleTooFewPrimaryNoExpertFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog *Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has not enough primary and no expert filters")

	currentDialogItem.AddToPath("notEnoughPrimaryNoExpertFilters")
	if !dialog.Previous().MatchesLastPath(PromptPrimaryPath) && !dialog.Previous().MatchesLastPath(PromptSecondaryPath) {
		emptyPrimaryFilters := currentDialogItem.InputFilter.GetEmptyPrimaryFilters()
		log.Debugf("not enough required filters prompted, will prompt additional primary filters %+v", emptyPrimaryFilters)

		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: emptyPrimaryFilters,
			},
		}
		currentDialogItem.AddToPath(PromptPrimaryPath)
		return currentDialogItem.OutputAction, nil
	}

	if dialog.Previous().MatchesLastPath(PromptPrimaryPath) {
		randomSecondaryFilters := currentDialogItem.InputFilter.GetRandomSecondaryFilters()
		log.Debugf("enough primary filters prompted or prompt attempts for primary filters are elapsed and no secondary filters prompted previously, will prompt random secondary filters %+v", randomSecondaryFilters)
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: randomSecondaryFilters,
			},
		}
		currentDialogItem.AddToPath(PromptSecondaryPath)
		return currentDialogItem.OutputAction, nil
	}

	currentDialogItem.AddToPathRecommend()
	log.Debug("prompted random secondary filters and enough primary filters or prompt attempts for primary filters are elapsed, decided to provide a recommendation")
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) handleNameFilter(ctx context.Context, currentDialogItem *DialogItem) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("since name filter is provided, deciding to recommend by name")

	currentDialogItem.AddToPath("notEmptyNameFilter").AddToPathRecommend()
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) handleAllPrimaryFiltersNoSecondaryFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog *Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	currentDialogItem.AddToPath("allPrimaryFiltersNoSecondaryFilters")

	shouldPromptSecondaryFilter := utils.GetRandomBoolean()
	if dialog.Previous().MatchesLastPath(PromptRandomSecondaryPath) || !shouldPromptSecondaryFilter {
		log.Debug("all primary filters provided, secondary filter was prompted, decided to provide a recommendation")
		currentDialogItem.AddToPathRecommend()
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeRecommendation,
		}

		return currentDialogItem.OutputAction, nil
	}

	currentDialogItem.AddToPath(PromptRandomSecondaryPath)
	randomSecondaryFilters := currentDialogItem.InputFilter.GetRandomSecondaryFilters()
	log.Debugf("all primary filters provided, randomply decided to prompt for random secondary filters %+v", randomSecondaryFilters)
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeFilterPrompt,
		Context: map[string]interface{}{
			FiltersContext: randomSecondaryFilters,
		},
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) handleAllPrimarySomeSecondaryAndSomeExpertFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has all primary, some secondary and some expert filters, decided to do a recommendation")

	currentDialogItem.AddToPath("allPrimarySomeSecondaryAndSomeExpertFilters").AddToPathRecommend()

	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) handleHasNotEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog *Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has not enough primary and some secondary and expert filters")

	currentDialogItem.AddToPath("notEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert")

	if !dialog.Previous().MatchesLastPath(PromptPrimaryPath) && !dialog.Previous().MatchesLastPath(PromptSecondaryPath) {
		emptyPrimaryFilters := currentDialogItem.InputFilter.GetEmptyPrimaryFilters()
		log.Debugf("primary filters were not prompted yet, will prompt filters %+v", emptyPrimaryFilters)
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: emptyPrimaryFilters,
			},
		}
		currentDialogItem.AddToPath(PromptPrimaryPath)
		return currentDialogItem.OutputAction, nil
	}

	if dialog.Previous().MatchesLastPath(PromptPrimaryPath) {
		randomSecondaryFilters := currentDialogItem.InputFilter.GetRandomSecondaryFilters()
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: randomSecondaryFilters,
			},
		}
		currentDialogItem.AddToPath(PromptSecondaryPath)
		log.Debugf("primary filters were prompted already, will prompt random secondary filters %+v", randomSecondaryFilters)
		return currentDialogItem.OutputAction, nil
	}

	currentDialogItem.AddToPathRecommend()
	log.Debug("primary and random secondary filters were prompted already, decided to do a recommendation")
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) handleSomePrimaryNoSecondarySomeExpertFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog *Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has some primary, no secondary and some expert filters")

	currentDialogItem.AddToPath("somePrimaryNoSecondarySomeExpertFilters")

	if !dialog.Previous().MatchesLastPath(PromptPrimaryPath) && !dialog.Previous().MatchesLastPath(PromptSecondaryPath) {
		emptyPrimaryFilters := currentDialogItem.InputFilter.GetEmptyPrimaryFilters()
		log.Debugf("primary filters were not prompted yet, will prompt filters %+v", emptyPrimaryFilters)
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: emptyPrimaryFilters,
			},
		}
		currentDialogItem.AddToPath(PromptPrimaryPath)
		return currentDialogItem.OutputAction, nil
	}

	if dialog.Previous().MatchesLastPath(PromptPrimaryPath) {
		randomSecondaryFilters := currentDialogItem.InputFilter.GetRandomSecondaryFilters()
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: randomSecondaryFilters,
			},
		}
		log.Debugf("primary filters were prompted already, will prompt random secondary filters %+v", randomSecondaryFilters)
		currentDialogItem.AddToPath(PromptSecondaryPath)
		return currentDialogItem.OutputAction, nil
	}

	log.Debug("primary and random secondary filters were prompted already, decided to do a recommendation")
	currentDialogItem.AddToPathRecommend()
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
	}

	return currentDialogItem.OutputAction, nil
}

func (dh DialogHandler) missingPrimaryFilters(f *WineFilter) string {
	missingFilters := []string{}
	if f.Color == "" {
		missingFilters = append(missingFilters, "цвет")
	}

	if f.Country == "" {
		missingFilters = append(missingFilters, "страна")
	}

	if len(f.Style) == 0 {
		missingFilters = append(missingFilters, "стиль")
	}

	return strings.Join(missingFilters, ", ")
}

func (dh DialogHandler) loadDialog(userId string) (*Dialog, error) {
	today := time.Now().UTC()
	dialogWindow := today.Add(ConversationHistoryWindow)

	dialg := &Dialog{}
	res := dh.db.
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userId, dialogWindow, today).
		Order("created_at DESC").
		Find(dialg)

	if res.Error != nil {
		return nil, errors.Wrap(res.Error)
	}

	return dialg, nil
}
