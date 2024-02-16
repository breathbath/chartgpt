package recommend

import (
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	logging "github.com/sirupsen/logrus"
	"gopkg.in/errgo.v2/errors"
	"gorm.io/gorm"
	"strings"
	"time"
)

const (
	ConversationHistoryWindow  = -time.Minute * 30
	ActionTypeFilterPrompt     = "filter_prompt"
	FiltersContext             = "filters"
	AdditionalTextPromptType   = "additional_text_prompt_type"
	PromptedPreviousLikedWines = "previous_liked_wines"
	ActionTypeRecommendation   = "recommendation"
	PathSeparator              = "->"
	RandPath                   = "rand"
	RecommendationPath         = ActionTypeRecommendation
	PromptSecondaryPath        = "promptSecondary"
	PromptPrimaryPath          = "promptPrimary"
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

func (a Action) IsPromptedPreviousLikedWines() bool {
	if _, ok := a.Context[AdditionalTextPromptType]; ok && a.Context[AdditionalTextPromptType] == PromptedPreviousLikedWines {
		return true
	}

	return false
}

func (a Action) GetAdditionalTextPromptType() string {
	additionalTextPromptType, ok := a.Context[AdditionalTextPromptType]
	if !ok {
		return ""
	}

	return fmt.Sprint(additionalTextPromptType)
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

func (d Dialog) LatestPromptsCount() int {
	previousPromptsCount := 0
	for _, dialogItem := range d {
		if dialogItem.OutputAction.Type == ActionTypeFilterPrompt {
			previousPromptsCount++
		} else {
			break
		}
	}

	return previousPromptsCount
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

	primaryFiltersCount := f.GetPrimaryFiltersCount()

	if primaryFiltersCount < f.GetTotalPrimaryFiltersCount() && primaryFiltersCount > 0 && f.HasSecondaryFilters() && f.HasExpertFilters() {
		return dh.handleHasNotEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert(ctx, currentDialogItem, dialog)
	}

	if primaryFiltersCount == f.GetTotalPrimaryFiltersCount() && f.HasSecondaryFilters() && f.HasExpertFilters() {
		return dh.handleAllPrimarySomeSecondaryAndSomeExpertFilters(ctx, currentDialogItem)
	}

	if primaryFiltersCount < f.GetTotalPrimaryFiltersCount() && primaryFiltersCount > 0 && !f.HasSecondaryFilters() && f.HasExpertFilters() {
		return dh.handleSomePrimaryNoSecondarySomeExpertFilters(ctx, currentDialogItem, dialog)
	}

	if primaryFiltersCount < f.GetTotalPrimaryFiltersCount() && !f.HasExpertFilters() {
		return dh.handleSomePrimaryNoExpertFilters(ctx, currentDialogItem, dialog)
	}

	return dh.handleHasAllPrimaryFilters(ctx, currentDialogItem, dialog)
}

func (dh DialogHandler) handleSomePrimaryNoExpertFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog Dialog,
) (*Action, error) {
	currentDialogItem.AddToPath("somePrimaryNoExpertFilters")
	primaryFiltersCount := currentDialogItem.InputFilter.GetPrimaryFiltersCount()
	if primaryFiltersCount == currentDialogItem.InputFilter.GetTotalPrimaryFiltersCount()-1 {
		return dh.handleAlmostAllPrimaryNoExpertFilters(ctx, currentDialogItem, dialog)
	}

	return dh.handleTooFewPrimaryNoExpertFilters(ctx, currentDialogItem, dialog)
}

func (dh DialogHandler) handleAlmostAllPrimaryNoExpertFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has almost all primary and no expert filters")

	currentDialogItem.AddToPath("almostAllPrimaryNoExpertFilters")

	previousMessage := dialog.Previous()
	if previousMessage == nil || previousMessage.OutputAction.Type != ActionTypeFilterPrompt {
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

	shouldPromptSecondaryFilters := utils.GetRandomBoolean()
	if shouldPromptSecondaryFilters {
		log.Debug("detected previous prompts, decided randomly to prompt secondary filters again")
		randomSecondaryFilters := currentDialogItem.InputFilter.GetRandomSecondaryFilters()
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: randomSecondaryFilters,
			},
		}
		currentDialogItem.AddToPathRandom().AddToPath(PromptSecondaryPath)
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
	dialog Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has not enough primary and no expert filters")

	primaryFiltersCount := currentDialogItem.InputFilter.GetPrimaryFiltersCount()

	currentDialogItem.AddToPath("notEnoughPrimaryNoExpertFilters")
	previousPromptsCount := dialog.LatestPromptsCount()
	const maxPreviosPrimaryFilterPromptsCount = 3
	if previousPromptsCount == 0 || (previousPromptsCount < maxPreviosPrimaryFilterPromptsCount && primaryFiltersCount < currentDialogItem.InputFilter.GetTotalPrimaryFiltersCount()-1) {
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

	previousPrompt := dialog.Previous()

	previousPromptIncludesSecondaryFilters := previousPrompt != nil && previousPrompt.InputFilter.IncludesSecondaryFilters(previousPrompt.OutputAction.GetFilters())
	if !previousPromptIncludesSecondaryFilters {
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

func (dh DialogHandler) handleHasAllPrimaryFilters(
	ctx context.Context,
	currentDialogItem *DialogItem,
	dialog Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	currentDialogItem.AddToPath("hasAllPrimaryFilters")

	previousPrompt := dialog.Previous()

	previousPromptIncludesSecondaryFilters := previousPrompt != nil && previousPrompt.InputFilter.IncludesSecondaryFilters(previousPrompt.OutputAction.GetFilters())
	if previousPromptIncludesSecondaryFilters {
		log.Debug("all required filters are prompted and previous prompt includes secondary filters, decided to give a recommendation")
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeRecommendation,
		}
		currentDialogItem.AddToPathRecommend()
		return currentDialogItem.OutputAction, nil
	}

	shouldPromptSecondaryFilter := utils.GetRandomBoolean()
	if shouldPromptSecondaryFilter {
		randomSecondaryFilters := currentDialogItem.InputFilter.GetRandomSecondaryFilters()
		log.Debugf("all primary filters provided, randomply decided to prompt for random secondary filters %+v", randomSecondaryFilters)
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				FiltersContext: randomSecondaryFilters,
			},
		}
		currentDialogItem.AddToPathRandom().AddToPath(PromptSecondaryPath)
		return currentDialogItem.OutputAction, nil
	}

	if previousPrompt != nil && previousPrompt.OutputAction.GetAdditionalTextPromptType() == PromptedPreviousLikedWines {
		log.Debugf("all primary filters provided and already prompted for liked wines, decided to provide a recommendation")
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeRecommendation,
		}

		currentDialogItem.AddToPathRecommend()
		return currentDialogItem.OutputAction, nil
	}

	shouldPromptPreviousLikedWines := utils.GetRandomBoolean()
	if shouldPromptPreviousLikedWines {
		log.Debug("all primary filters provided, randomly decided to about previously liked wines")
		currentDialogItem.OutputAction = &Action{
			Type: ActionTypeFilterPrompt,
			Context: map[string]interface{}{
				AdditionalTextPromptType: PromptedPreviousLikedWines,
			},
		}
		currentDialogItem.AddToPathRandom().AddToPath("previouslyLikedWines")
		return currentDialogItem.OutputAction, nil
	}

	log.Debug("all primary filters provided, randomly decided not to prompt previously liked wines, decided to provide a recommendation")
	currentDialogItem.AddToPathRecommend()
	currentDialogItem.OutputAction = &Action{
		Type: ActionTypeRecommendation,
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
	dialog Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has not enough primary and some secondary and expert filters")

	currentDialogItem.AddToPath("notEnoughPrimaryFiltersAndSomeSecondaryAndSomeExpert")

	previousMessage := dialog.Previous()
	if previousMessage == nil || previousMessage.OutputAction.Type != ActionTypeFilterPrompt {
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

	if previousMessage.OutputAction.Type == ActionTypeFilterPrompt &&
		previousMessage.InputFilter.IncludesPrimaryFilters(previousMessage.OutputAction.GetFilters()) {
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
	dialog Dialog,
) (*Action, error) {
	log := logging.WithContext(ctx)
	log.Debug("has some primary, no secondary and some expert filters")

	currentDialogItem.AddToPath("somePrimaryNoSecondarySomeExpertFilters")

	previousMessage := dialog.Previous()
	if previousMessage == nil {
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

	if previousMessage.OutputAction.Type == ActionTypeFilterPrompt && previousMessage.InputFilter.IncludesPrimaryFilters(previousMessage.OutputAction.GetFilters()) {
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

func (dh DialogHandler) loadDialog(userId string) (Dialog, error) {
	today := time.Now().UTC()
	dialogWindow := today.Add(ConversationHistoryWindow)

	dialg := Dialog{}
	res := dh.db.
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userId, dialogWindow, today).
		Order("created_at DESC").
		Find(&dialg)

	if res.Error != nil {
		return nil, errors.Wrap(res.Error)
	}

	return dialg, nil
}
