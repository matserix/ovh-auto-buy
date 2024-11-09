package main

/** 
 * Source code modified from: https://blog.yessure.org/index.php/archives/203
 */

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/ovh/go-ovh/ovh"
)

var (
    appKey      = os.Getenv("APP_KEY")			// OVH的应用key
    appSecret   = os.Getenv("APP_SECRET")		// OVH的应用secret
    consumerKey = os.Getenv("CONSUMER_KEY")		// OVH的消费者key
    region      = os.Getenv("REGION")			// 区域设置为, e.g. ovh-eu
    tgtoken     = os.Getenv("TG_TOKEN")			// 你的Telegram Bot Token
    tgchatid    = os.Getenv("TG_CHATID")		// 你希望发送消息的Telegram Chat ID
    zone        = os.Getenv("ZONE")				// OVH子公司区域设置, e.g. IE
    plancode    = os.Getenv("PLANCODE")         // 需要购买的产品的planCode, e.g. 25skleb01
    optionsenv  = os.Getenv("OPTIONS")          // 选择的配置用逗号分隔, e.g. bandwidth-300-25skle,ram-32g-ecc-2400-25skle,softraid-2x450nvme-25skle
    autopay     = os.Getenv("AUTOPAY")          // 是否自动支付, e.g. true
	frequency	= os.Getenv("FREQUENCY")		// 检查频率单位为秒, e.g. 5
)

var bought = false                              // 是否已购买, 避免os.Exit(0)出错

func runTask() {

    client, err := ovh.NewClient(region, appKey, appSecret, consumerKey)
    if err != nil {
        log.Printf("Failed to create OVH client: %v\n", err)
        return
    }

    var result []map[string]interface{}
    err = client.Get("/dedicated/server/datacenter/availabilities", &result)
    if err != nil {
        log.Printf("Failed to get datacenter availabilities: %v\n", err)
        return
    }

    foundAvailable := false
    var fqn, planCode, datacenter string

    for _, item := range result {
        if item["planCode"] == plancode {
            fqn = item["fqn"].(string)
            planCode = item["planCode"].(string)
            datacenters := item["datacenters"].([]interface{})

            for _, dcInfo := range datacenters {
                dc := dcInfo.(map[string]interface{})
                availability := dc["availability"].(string)
                datacenter = dc["datacenter"].(string)

                fmt.Printf("FQN: %s\n", fqn)
                fmt.Printf("Availability: %s\n", availability)
                fmt.Printf("Datacenter: %s\n", datacenter)
                fmt.Println("------------------------")

                if availability != "unavailable" {
                    foundAvailable = true
                    break
                }
            }

            if foundAvailable {
                fmt.Printf("Proceeding to next step with FQN: %s Datacenter: %s\n", fqn, datacenter)
                break
            }
        }
    }

    if !foundAvailable {
        log.Println("No record to buy")
        return
    }

    msg_available := fmt.Sprintf("🔥 有货: %s 在 %s 地区", plancode, datacenter)
    sendTelegramMsg(tgtoken, tgchatid, msg_available)

    fmt.Println("Create cart")
    var cartResult map[string]interface{}
    err = client.Post("/order/cart", map[string]interface{}{
        "ovhSubsidiary": zone,
    }, &cartResult)
    if err != nil {
        log.Printf("Failed to create cart: %v\n", err)
        return
    }

    cartID := cartResult["cartId"].(string)
    fmt.Printf("Cart ID: %s\n", cartID)

    fmt.Println("Assign cart")
    err = client.Post("/order/cart/"+cartID+"/assign", nil, nil)
    if err != nil {
        log.Printf("Failed to assign cart: %v\n", err)
        return
    }

    fmt.Println("Put item into cart")
    var itemResult map[string]interface{}
    err = client.Post("/order/cart/"+cartID+"/eco", map[string]interface{}{
        "planCode":    planCode,
        "pricingMode": "default",
        "duration":    "P1M",
        "quantity":    1,
    }, &itemResult)
    if err != nil {
        log.Printf("Failed to add item to cart: %v\n", err)
        return
    }

    var itemID string
    if v, ok := itemResult["itemId"].(json.Number); ok {
        itemID = v.String()
    } else if v, ok := itemResult["itemId"].(string); ok {
        itemID = v
    } else {
        log.Printf("Unexpected type for itemId, expected json.Number or string, got %T\n", itemResult["itemId"])
        return
    }

    fmt.Printf("Item ID: %s\n", itemID)

    fmt.Println("Checking required configuration")
    var requiredConfig []map[string]interface{}
    err = client.Get("/order/cart/"+cartID+"/item/"+itemID+"/requiredConfiguration", &requiredConfig)
    if err != nil {
        log.Printf("Failed to get required configuration: %v\n", err)
        return
    }

    dedicatedOs := "none_64.en"
    var regionValue string
    for _, config := range requiredConfig {
        if config["label"] == "region" {
            if allowedValues, ok := config["allowedValues"].([]interface{}); ok && len(allowedValues) > 0 {
                regionValue = allowedValues[0].(string)
            }
        }
    }

    configurations := []map[string]interface{}{
        {"label": "dedicated_datacenter", "value": datacenter},
        {"label": "dedicated_os", "value": dedicatedOs},
        {"label": "region", "value": regionValue},
    }

    for _, config := range configurations {
        fmt.Printf("Configure %s\n", config["label"])
        err = client.Post("/order/cart/"+cartID+"/item/"+itemID+"/configuration", map[string]interface{}{
            "label": config["label"],
            "value": config["value"],
        }, nil)
        if err != nil {
            log.Printf("Failed to configure %s: %v\n", config["label"], err)
            return
        }
    }

    fmt.Println("Add options")
    options := strings.Split(optionsenv, ",")

    itemIDInt, _ := strconv.Atoi(itemID)
    for _, option := range options {
        err = client.Post("/order/cart/"+cartID+"/eco/options", map[string]interface{}{
            "duration":    "P1M",
            "itemId":      itemIDInt,
            "planCode":    option,
            "pricingMode": "default",
            "quantity":    1,
        }, nil)
        if err != nil {
            log.Printf("Failed to add option %s: %v\n", option, err)
            return
        }
    }

    fmt.Println("Checkout")
    var checkoutResult map[string]interface{}
    err = client.Get("/order/cart/"+cartID+"/checkout", &checkoutResult)
    if err != nil {
        log.Printf("Failed to get checkout: %v\n", err)
        return
    }

    autopayValue, err := strconv.ParseBool(autopay)
    if err != nil {
        fmt.Println("AUTOPAY value is invalid:", err)
        return
    }

    err = client.Post("/order/cart/"+cartID+"/checkout", map[string]interface{}{
        "autoPayWithPreferredPaymentMethod": autopayValue,
        "waiveRetractationPeriod":           true,
    }, nil)
    if err != nil {
        log.Printf("Failed to checkout: %v\n", err)
        return
    }
    log.Println("Ordered!")
	bought = true
    msg_ordered := fmt.Sprintf("🎉 订购成功: %s 在 %s 地区", datacenter, plancode)
    sendTelegramMsg(tgtoken, tgchatid, msg_ordered)
    os.Exit(0)
}

func sendTelegramMsg(botToken, chatID, message string) error {
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
    payload := map[string]string{
        "chat_id": chatID,
        "text":    message,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("error encoding JSON: %v", err)
    }

    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("error sending request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("received non-OK response status: %v", resp.Status)
    }

    return nil
}

func main() {
    freq, err := strconv.Atoi(frequency)
    if err != nil {
        fmt.Println("Error converting frequency:", err)
        return
    }
    for {
        if (bought == false) {
			runTask()
		}
        time.Sleep(time.Duration(freq) * time.Second)
    }
}