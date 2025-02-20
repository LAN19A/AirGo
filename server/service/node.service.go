package service

import (
	"AirGo/global"
	"AirGo/model"
	"gorm.io/gorm/clause"
	"strconv"
	"time"
)

// 根据node name 模糊查询节点
func GetNodeByName(remarks string) ([]model.Node, error) {
	var nodes []model.Node
	err := global.DB.Where("remarks like ?", ("%" + remarks + "%")).Order("node_order").Find(&nodes).Error
	return nodes, err
}

// 查询全部节点
func GetAllNode() (*[]model.Node, error) {
	var nodes []model.Node
	err := global.DB.Order("node_order").Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return &nodes, nil
}

// 新建节点
func NewNode(node *model.Node) error {
	//node.ID = 0 //清空节点id 防止插入失败
	err := global.DB.Create(&node).Error
	return err
}

// 删除节点
func DeleteNode(node *model.Node) error {
	//删除关联
	err := global.DB.Where("node_id = ?", node.ID).Delete(&model.GoodsAndNodes{}).Error
	if err != nil {
		return err
	}
	err = global.DB.Where(&model.Node{ID: node.ID}).Delete(&model.Node{}).Error
	return err
}

// 更新节点
func UpdateNode(node *model.Node) error {
	return global.DB.Save(&node).Error
}

// 查询节点流量
func GetNodeTraffic(params model.QueryParamsWithDate) model.NodesWithTotal {
	var nodesWithTotal model.NodesWithTotal
	var startTime, endTime time.Time
	//时间格式转换
	if len(params.Date) == 2 {
		startTime, _ = time.ParseInLocation("2006-01-02 15:04:05", params.Date[0], time.Local)
		endTime, _ = time.ParseInLocation("2006-01-02 15:04:05", params.Date[1], time.Local)
	} else {
		//默认前1个月数据
		endTime = time.Now().Local()
		startTime = endTime.AddDate(0, 0, -30)
	}
	if params.Search != "" {
		err := global.DB.Model(&model.Node{}).Count(&nodesWithTotal.Total).Where("remarks LIKE ?", "%"+params.Search+"%").Limit(int(params.PageSize)).Offset((int(params.PageNum)-1)*int(params.PageSize)).Preload("TrafficLogs", global.DB.Where("created_at > ? and created_at < ?", startTime, endTime)).Order("node_order").Find(&nodesWithTotal.NodeList).Error
		if err != nil {
			global.Logrus.Error("查询节点流量error:", err.Error())
			return model.NodesWithTotal{}
		}
	} else {
		err := global.DB.Model(&model.Node{}).Count(&nodesWithTotal.Total).Limit(int(params.PageSize)).Offset((int(params.PageNum)-1)*int(params.PageSize)).Preload("TrafficLogs", global.DB.Where("created_at > ? and created_at < ?", startTime, endTime)).Order("node_order").Find(&nodesWithTotal.NodeList).Error
		if err != nil {
			global.Logrus.Error("查询节点流量error:", err.Error())
			return model.NodesWithTotal{}
		}
	}
	for i1, _ := range nodesWithTotal.NodeList {
		for _, v := range nodesWithTotal.NodeList[i1].TrafficLogs {
			nodesWithTotal.NodeList[i1].TotalUp = nodesWithTotal.NodeList[i1].TotalUp + v.U
			nodesWithTotal.NodeList[i1].TotalDown = nodesWithTotal.NodeList[i1].TotalDown + v.D
		}
		//nodeArr[i1].TrafficLogs=[]model.TrafficLog{} //清空traffic
	}
	return nodesWithTotal
}

// 获取 node status，用于探针
func GetNodesStatus() *[]model.NodeStatus {
	var nodesIds []model.Node
	global.DB.Model(&model.Node{}).Select("id", "remarks", "traffic_rate").Order("node_order").Find(&nodesIds)
	var nodestatusArr []model.NodeStatus
	for _, v := range nodesIds {
		var nodeStatus = model.NodeStatus{}
		vStatus, ok := global.LocalCache.Get(strconv.FormatInt(v.ID, 10) + "status")
		if !ok { //cache过期，离线了
			nodeStatus.ID = v.ID
			nodeStatus.Name = v.Remarks
			nodeStatus.TrafficRate = v.TrafficRate
			nodeStatus.Status = false
			nodeStatus.D = 0
			nodeStatus.U = 0
			nodestatusArr = append(nodestatusArr, nodeStatus)
		} else {
			nodeStatus = vStatus.(model.NodeStatus)
			nodeStatus.Name = v.Remarks
			nodeStatus.TrafficRate = v.TrafficRate
			nodestatusArr = append(nodestatusArr, nodeStatus)
		}
	}
	return &nodestatusArr
}

// 插入节点流量统计
func NewTrafficLog(t *model.TrafficLog) error {
	return global.DB.Create(&t).Error
}

// 定时清理数据库(traffic)
func CleanDBTraffic() error {
	y, m, _ := time.Now().Date()
	startTime := time.Date(y, m-2, 1, 0, 0, 0, 0, time.Local)
	return global.DB.Where("created_at < ?", startTime).Delete(&model.TrafficLog{}).Error
}

// 节点排序
func NodeSort(nodeArr *[]model.Node) error {
	return global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"node_order"}),
	}).Create(&nodeArr).Error
}
