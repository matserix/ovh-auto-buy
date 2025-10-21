# Automatically buys OVH Eco Servers (Docker)
Go source code modified from: https://blog.yessure.org/index.php/archives/203

To generate token:
- IE: `https://eu.api.ovh.com/createToken/index.cgi?GET=/*&PUT=/*&POST=/*&DELETE=/*`
- CA: `https://ca.api.ovh.com/createToken/index.cgi?GET=/*&PUT=/*&POST=/*&DELETE=/*`

To get Eco planCode and options:
- IE: https://eu.api.ovh.com/v1/order/catalog/public/eco?ovhSubsidiary=IE
- CA: https://ca.api.ovh.com/v1/order/catalog/public/eco?ovhSubsidiary=CA

## Docker
```bash
docker run -e APP_KEY="ovh_appkey" \
           -e APP_SECRET="ovh_appsecret" \
           -e CONSUMER_KEY="ovh_consumerkey" \
           -e REGION="ovh_region" \
           -e TG_TOKEN="telegram_bot_token" \
           -e TG_CHATID="telegram_your_chatid" \
           -e ZONE="ovh_zone" \
           -e PLANCODE="product_plancode" \
           -e OPTIONS="product_options" \
           -e AUTOPAY=true \
           -e FREQUENCY=5 \
           katorly/ovh-auto-buy:latest
```

## docker-compose.yml
```yaml
services:
  auto-buy:
    image: katorly/ovh-auto-buy:latest
    container_name: ks-buy
    environment:
      APP_KEY: "ovh_appkey"                          # OVH Application Key
      APP_SECRET: "ovh_appsecret"                    # OVH Application Secret
      CONSUMER_KEY: "ovh_consumerkey"                # OVH Consumer Key
      REGION: "ovh_region"                           # Region setting, e.g., ovh-eu
      TG_TOKEN: "telegram_bot_token"                 # Your Telegram Bot Token
      TG_CHATID: "telegram_your_chatid"              # The Telegram Chat ID where you want to send messages
      ZONE: "ovh_zone"                               # OVH subsidiary region setting, e.g., IE
      PLANCODE: "product_plancode"                   # The planCode for the product you need to purchase, e.g., 25skleb01
      FQN: "fqn"                                     # The FQN for  the product (will be used instead of planCode)
      OPTIONS: "product_options"                     # Selected configurations, comma-separated, e.g., bandwidth-300-25skle,ram-32g-ecc-2400-25skle,softraid-2x450nvme-25skle
      AUTOPAY: true                                  # Whether to enable autopay, e.g., true
      SKIPPED_DATACENTERS: "skipped_datacenters"     # Datacenters to skip, comma-separated, e.g., bhs,gra
      CHECK_CATALOG: true                            # Whether to check the catalog, e.g., true
      FREQUENCY: 5                                   # Check frequency in seconds, e.g., 5
```
