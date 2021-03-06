package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/solefaucet/sole-server/models"
	"github.com/solefaucet/sole-server/utils"
)

// GetReward randomly gives users reward
func GetReward(
	getUserByID dependencyGetUserByID,
	getLatestTotalReward dependencyGetLatestTotalReward,
	getSystemConfig dependencyGetSystemConfig,
	getRewardRatesByType dependencyGetRewardRatesByType,
	createRewardIncome dependencyCreateRewardIncome,
	cacheIncome dependencyInsertIncome,
	broadcast dependencyBroadcast,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := c.MustGet("auth_token").(models.AuthToken)
		now := time.Now()

		// get user
		user, err := getUserByID(authToken.UserID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// check last rewarded time
		if user.RewardedAt.Add(time.Second * time.Duration(user.RewardInterval)).After(now) {
			c.AbortWithStatus(statusCodeTooManyRequests)
			return
		}

		// get random reward
		config := getSystemConfig()
		latestTotalReward := getLatestTotalReward()
		rewardRateType := models.RewardRateTypeLess
		if latestTotalReward.IsSameDay(now) && latestTotalReward.Total > config.TotalRewardThreshold {
			rewardRateType = models.RewardRateTypeMore
		}
		rewardRates := getRewardRatesByType(rewardRateType)
		reward := utils.RandomReward(rewardRates)
		rewardReferer := reward * config.RefererRewardRate

		// double reward if needed
		doubled := config.DoubleToday()
		if doubled {
			reward *= 2
		}

		// create income reward
		income := models.Income{
			UserID:        user.ID,
			RefererID:     user.RefererID,
			Type:          models.IncomeTypeReward,
			Income:        reward,
			RefererIncome: rewardReferer,
		}
		if err := createRewardIncome(income, now); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// cache delta income
		deltaIncome := struct {
			Address string    `json:"address"`
			Amount  float64   `json:"amount"`
			Type    string    `json:"type"`
			Time    time.Time `json:"time"`
		}{user.Address, reward, "reward", now}
		cacheIncome(deltaIncome)

		// broadcast delta income to all clients
		msg, _ := json.Marshal(models.WebsocketMessage{DeltaIncome: deltaIncome})
		broadcast(msg)

		referer, _ := getUserByID(user.RefererID)
		logrus.WithFields(logrus.Fields{
			"event":            models.EventReward,
			"user_email":       user.Email,
			"user_address":     user.Address,
			"user_ip":          c.ClientIP(),
			"user_rewarded_at": user.RewardedAt,
			"referer_email":    referer.Email,
			"reward_rate_type": rewardRateType,
			"amount":           reward,
			"reward_doubled":   doubled,
		}).Info("user get reward")

		c.JSON(http.StatusOK, income)
	}
}

// RewardList returns user's reward list as response
func RewardList(
	getRewardIncomes dependencyGetRewardIncomes,
	getNumberOfRewardIncomes dependencyGetNumberOfRewardIncomes,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken := c.MustGet("auth_token").(models.AuthToken)

		// parse pagination args
		limit, offset, err := parsePagination(c)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		rewards, err := getRewardIncomes(authToken.UserID, limit, offset)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		count, err := getNumberOfRewardIncomes(authToken.UserID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, paginationResult(rewards, count))
	}
}
