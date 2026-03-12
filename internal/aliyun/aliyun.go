package aliyun

import (
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	"security-group/internal/config"
)

const descriptionPrefix = "auto-sg:"

type Client struct {
	client          *ecs.Client
	securityGroupID string
}

func New(cfg *config.AliyunConfig) (*Client, error) {
	client, err := ecs.NewClientWithAccessKey(cfg.RegionID, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("create aliyun client: %w", err)
	}
	return &Client{
		client:          client,
		securityGroupID: cfg.SecurityGroupID,
	}, nil
}

type Rule struct {
	IpProtocol   string
	PortRange    string
	SourceCidrIp string
	Description  string
}

func (c *Client) QueryRules(username string) ([]Rule, error) {
	req := ecs.CreateDescribeSecurityGroupAttributeRequest()
	req.SecurityGroupId = c.securityGroupID
	req.Direction = "ingress"

	resp, err := c.client.DescribeSecurityGroupAttribute(req)
	if err != nil {
		return nil, fmt.Errorf("query security group: %w", err)
	}

	desc := descriptionPrefix + username
	var rules []Rule
	for _, p := range resp.Permissions.Permission {
		if p.Description == desc {
			rules = append(rules, Rule{
				IpProtocol:   p.IpProtocol,
				PortRange:    p.PortRange,
				SourceCidrIp: p.SourceCidrIp,
				Description:  p.Description,
			})
		}
	}
	return rules, nil
}

func (c *Client) AddRule(ip, username string) error {
	req := ecs.CreateAuthorizeSecurityGroupRequest()
	req.SecurityGroupId = c.securityGroupID
	req.IpProtocol = "all"
	req.PortRange = "-1/-1"
	req.SourceCidrIp = ip + "/32"
	req.Description = descriptionPrefix + username

	_, err := c.client.AuthorizeSecurityGroup(req)
	if err != nil {
		return fmt.Errorf("add security group rule: %w", err)
	}
	return nil
}

func (c *Client) RemoveRule(rule Rule) error {
	req := ecs.CreateRevokeSecurityGroupRequest()
	req.SecurityGroupId = c.securityGroupID
	req.IpProtocol = rule.IpProtocol
	req.PortRange = rule.PortRange
	req.SourceCidrIp = rule.SourceCidrIp

	_, err := c.client.RevokeSecurityGroup(req)
	if err != nil {
		return fmt.Errorf("remove security group rule: %w", err)
	}
	return nil
}

func (c *Client) UpdateIP(ip, username string) (string, error) {
	rules, err := c.QueryRules(username)
	if err != nil {
		return "", err
	}

	targetCidr := ip + "/32"

	for _, r := range rules {
		if r.SourceCidrIp == targetCidr {
			return "IP 未变化，无需更新", nil
		}
	}

	for _, r := range rules {
		if err := c.RemoveRule(r); err != nil {
			return "", fmt.Errorf("删除旧规则失败: %w", err)
		}
	}

	if err := c.AddRule(ip, username); err != nil {
		return "", fmt.Errorf("添加新规则失败: %w", err)
	}

	return "安全组规则已更新", nil
}
