package worker

import (
	"errors"
	"fmt"
	"math/rand"
	"metric-index/conf"
	"metric-index/dao/gocache"
	"metric-index/dao/mysql"
	"metric-index/dao/redis"
	"metric-index/services/metrics"
	"time"

	"go.uber.org/zap"
)

// Store 存储metric信息到mysql，组装sql，执行sql
func Store(metric *metrics.Metric) {
	var (
		metricID     int
		basicLabelID int
		err          error
	)

	// 获取metricBasicLabelID，通过查询缓存，或者插入数据库后再查询数据，获取id
	basicLabelID, err = getOrCreateBasicLabelID(metric)
	if err != nil {
		zap.L().Error("get basicLabelID err",
			zap.Any("metric", metric),
			zap.Error(err))
		return
	}

	// 获取metricID
	metricID, err = getOrCreateMetricID(metric, basicLabelID)
	if err != nil {
		zap.L().Error("getOrCreateMetricID err",
			zap.Any("metric", metric),
			zap.Error(err))
		return
	}

	// 将除去basicLabel的其他label保存到数据库中
	if err := saveMetricLabel(metric, metricID); err != nil {
		zap.L().Error("saveMetricLabel err",
			zap.Any("metric", metric),
			zap.Error(err))
		return
	}

}

// get or create basicLabelID
func getOrCreateBasicLabelID(metric *metrics.Metric) (basicLabelID int, err error) {
	var (
		env        string
		service    string
		metricName string
	)

	for i := range metric.Labels {
		switch metric.Labels[i].Name {
		case "env":
			env = metric.Labels[i].Value
		case "service":
			service = metric.Labels[i].Value
		case "__name__":
			metricName = metric.Labels[i].Value
		}
	}

	if _, err := validateBasicLabel(env, service, metricName); err != nil {
		return 0, err
	}

	cacheKey := fmt.Sprintf("env:%s,service:%s,name:%s", env, service, metricName)
	basicLabelID, err = getIDFromCache(cacheKey)
	if err != nil {
		saveBasicLabelSQL := fmt.Sprintf("INSERT IGNORE INTO `metric_basic_label` (`env`, `service`, `__name__`) VALUES ('%s', '%s', '%s')", env, service, metricName)
		getBasicLabelIDSQL := fmt.Sprintf("SELECT `id` FROM `metric_basic_label` WHERE `env`='%s' AND `service`='%s' AND `__name__`='%s'", env, service, metricName)
		if err := mysql.Exec(saveBasicLabelSQL); err != nil {
			zap.L().Error("insert basicLabel into mysql err",
				zap.String("saveMetricBasicLabelSQL", saveBasicLabelSQL),
				zap.Error(err))
		}
		basicLabelID, err = mysql.GetID(getBasicLabelIDSQL)
		if err != nil {
			zap.L().Error("get metricBasicLabelID err",
				zap.String("getMetricBasicLabelIDSQL", getBasicLabelIDSQL),
				zap.Error(err))
			return
		}
		send2Cache(cacheKey, basicLabelID)
	}

	return
}

// 获取metircID，如果获取是在，则先保存到db中，然后再get出id
func getOrCreateMetricID(metric *metrics.Metric, basicLabelID int) (metricID int, err error) {
	metricID, err = getIDFromCache(metric.Content)
	if err != nil {
		saveMetricSQL := fmt.Sprintf("INSERT IGNORE INTO `metric` (`metric_basic_label_id`, `metric`) VALUES ('%d', '%s')", basicLabelID, metric.Content)
		getMetricIDSQL := fmt.Sprintf("SELECT `id` FROM `metric` WHERE `metric`='%s'", metric.Content)
		if err = mysql.Exec(saveMetricSQL); err != nil {
			zap.L().Error("insert metric into mysql err",
				zap.String("saveMetricSQL", saveMetricSQL),
				zap.Error(err))
		}

		metricID, err = mysql.GetID(getMetricIDSQL)
		if err != nil {
			zap.L().Error("get labelNameID err",
				zap.String("getMetricIDSQL", getMetricIDSQL),
				zap.Error(err))
			return
		}

		send2Cache(metric.Content, metricID)
	}
	return metricID, nil
}

// 遍历labels，将除env、service、__name__ 之外的label插入到数据库中
func saveMetricLabel(metric *metrics.Metric, metricID int) error {
	var (
		labelNameID  int
		labelValueID int
		err          error
	)
	// 创建插入labelName或labelValue的SQL
	labelNameSQLFmt := "INSERT IGNORE INTO `metric_label_name` (`name`) VALUES ('%s')"
	labelValueSQLFmt := "INSERT IGNORE INTO `metric_label_value` (`value`) VALUES ('%s')"
	metricLabelSQLFmt := "INSERT IGNORE INTO `metric_label` " +
		"(`metric_id`, `metric_label_name_id`, `metric_label_value_id`) VALUES ('%d', '%d', '%d')"
	getLabelNameIDSQLFmt := "SELECT `id` FROM `metric_label_name` WHERE `name`='%s'"
	getLabelValueIDSQLFmt := "SELECT `id` FROM `metric_label_value` WHERE `value`='%s'"

	// 遍历labels，将除env、service、__name__ 之外的label保存到数据库中
	for i := range metric.Labels {
		if metric.Labels[i].Name == "env" || metric.Labels[i].Name == "service" || metric.Labels[i].Name == "__name__" {
			continue
		}

		var (
			saveLabelNameSQL   string
			saveLabelValueSQL  string
			saveMetricLabelSQL string
			getLabelNameIDSQL  string
			getLabelValueIDSQL string
		)

		// get or create labelNameID
		labelNameID, err = getIDFromCache(metric.Labels[i].Name)
		if err != nil {
			saveLabelNameSQL = fmt.Sprintf(labelNameSQLFmt, metric.Labels[i].Name)
			getLabelNameIDSQL = fmt.Sprintf(getLabelNameIDSQLFmt, metric.Labels[i].Name)
			if err = mysql.Exec(saveLabelNameSQL); err != nil {
				zap.L().Error("insert labelName into mysql err",
					zap.String("saveLabelNameSQL", saveLabelNameSQL),
					zap.Error(err))
			}
			labelNameID, err = mysql.GetID(getLabelNameIDSQL)
			if err != nil {
				zap.L().Error("get labelNameID err",
					zap.String("getLabelNameIDSQL", getLabelNameIDSQL),
					zap.Error(err))
				return err
			}
			send2Cache(metric.Labels[i].Name, labelNameID)
		}

		// get or create labelValueID
		labelValueID, err = getIDFromCache(metric.Labels[i].Value)
		if err != nil {
			saveLabelValueSQL = fmt.Sprintf(labelValueSQLFmt, metric.Labels[i].Value)
			getLabelValueIDSQL = fmt.Sprintf(getLabelValueIDSQLFmt, metric.Labels[i].Value)
			if err := mysql.Exec(saveLabelValueSQL); err != nil {
				zap.L().Error("insert labelValue into mysql err",
					zap.String("saveLabelValueSQL", saveLabelValueSQL),
					zap.Error(err))
			}
			labelValueID, err = mysql.GetID(getLabelValueIDSQL)
			if err != nil {
				zap.L().Error("get labelValueID err",
					zap.String("getLabelValueIDSQL", getLabelValueIDSQL),
					zap.Error(err))
				return err
			}
			send2Cache(metric.Labels[i].Value, labelValueID)
		}

		// 验证三个ID是否都合法
		if valid, _ := validateID(metricID, labelNameID, labelValueID); !valid {
			zap.L().Error("id is not valid",
				zap.Int("metricID", metricID),
				zap.Int("labelNameID", labelNameID),
				zap.Int("labelValueID", labelValueID))
			return err
		}

		saveMetricLabelSQL = fmt.Sprintf(metricLabelSQLFmt,
			metricID,
			labelNameID,
			labelValueID)
		if err := mysql.Exec(saveMetricLabelSQL); err != nil {
			zap.L().Error("insert metriclabel into mysql err",
				zap.String("saveMetricLabelSQL:", saveMetricLabelSQL),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func send2Cache(key string, value int) {
	if value == 0 {
		zap.L().Error("wrong value, ID value is 0")
		return
	}
	// 添加到go缓存
	gocache.SetDefault(key, value)

	// 添加到reids缓存
	var exTime time.Duration
	if conf.Conf.MetricStore.Cache.IsExpire {
		exTime = time.Duration(conf.Conf.MetricStore.Cache.Expire-
			conf.Conf.MetricStore.Cache.DistInterval+
			rand.Intn(conf.Conf.MetricStore.Cache.DistInterval)) *
			time.Second
	} else {
		exTime = time.Duration(-1) * time.Second
	}
	if err := redis.Push(redis.CommitItem{
		Key:    key,
		Value:  value,
		ExTime: exTime,
	}); err != nil {
		zap.L().Error("push metric to redis committer failed", zap.Error(err))
	}
}

func getIDFromCache(key string) (ID int, err error) {
	id, found := gocache.Get(key)
	var ok bool
	if found {
		ID, ok = id.(int)
		if !ok {
			errStr := fmt.Sprintf("can not assert id: %v to int", id)
			err = errors.New(errStr)
			return 0, err
		}

		if ID == 0 {
			err = errors.New("wrong value, ID value is 0")
			return 0, err
		}
		return ID, nil
	}

	id, err = redis.Get(key)
	if err != nil {
		return 0, err
	}
	ID, ok = id.(int)
	if !ok {
		errStr := fmt.Sprintf("can not assert id: %v to int", id)
		err = errors.New(errStr)
		return 0, err
	}
	if ID == 0 {
		err = errors.New("wrong value, ID value is 0")
		return 0, err
	}
	gocache.SetDefault(key, ID)
	return ID, nil
}

// 验证id是否合法
func validateID(metricID, labelNameID, labelValueID int) (bool, error) {
	if metricID <= 0 || labelNameID <= 0 || labelValueID <= 0 {
		return false, nil
	}
	return true, nil
}

// 验证label value是否合法
func validateBasicLabel(env, service, metircName string) (valid bool, err error) {
	var errStr string
	valid = true
	switch "" {
	case env:
		errStr = "env is required"
		valid = false
	case service:
		errStr = "service is required"
		valid = false
	case metircName:
		errStr = "metricName is required"
		valid = false
	}
	if !valid {
		err = errors.New(errStr)
	}
	return
}
