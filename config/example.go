package config

import "strings"

var ExampleYaml = `
devices:
  light.kitchen:
    group: downstairs
    name: Kitchen
    type: light
    caps: [switch]
    source: x10.b06
    location: Kitchen
  light.glowworm:
    group: downstairs
    name: Glowworm
    type: light
    caps: [light, dimmer]
    source: homeeasy.00123453
    aliases: [glow worm]
    location: Living Room
  thermostat.living:
    name: Living room thermostat
    group: heating
    location: Living Room
  trv.living:
    name: Living room thermostat
    source: energenie.00097f
  trv.spareroom:
    name: Spareroom thermostat
    source: energenie.00098b
    location: Spareroom
endpoints:
  mqtt:
    broker: tcp://127.0.0.1:1883
processes:
  nonexistent:
    cmd: gohome service nonexistent
bill:
  electricity:
    primary_rate: 8.54
    standing_charge: 18.9
  gas:
    calorific_value: 39.3
    conversion_factor: 1.02264
    multiplier: 0.01
    primary_rate: 3.16
    standing_charge: 35.4
  vat: 0
  currency: £
earth:
  latitude: 51.5072
  longitude: 0.1275
espeak:
  args: -s 140
general:
  email:
    admin: admin@example.com
    from: me@example.com
    server: localhost:25
heating:
  device: heater.boiler
  hallway:
    schedule:
      Weekdays:
      - 00:00-23:59: 0
      Weekends:
      - 09:00-22:30: 15.5
  living:
    schedule:
      Friday:
      - 07:45-08:10: 16
      Monday,Tuesday,Wednesday,Thursday:
      - 07:40-08:10: 16.5
      Weekends:
      - 09:00-22:50: 16
  office:
    schedule:
      Weekdays:
      - 09:00-18:00: 16
      Saturday,Sunday:
      - 00:00-23:59: 0
  minimum: 10
  slop: 0.3
irrigation:
  at: 6h
  device: pump.garden
  enabled: true
  factor: 1.5
  interval: 12h
  max_temp: 25
  max_time: 60
  min_temp: 13
  min_time: 10
jabber:
  jid: myjabberid@gmail.com/gohome
  pass: password
sms:
  telephone: '+441234567890'
twitter:
  auth:
    consumer_key: xxx
    consumer_secret: yyy
    token: aaa
    token_secret: bbb
voice:
  'lights? on':
    switch glowworm on
  'lights? off':
    switch glowworm off
  'blanket on':
    switch blanket on
  'blanket off':
    switch blanket off
watchdog:
  devices:
    power.power: 5m
    temp.garden: 20m
    temp.hallway: 4h
    temp.living: 20m
    temp.office: 20m
    temp.outside: 20m
    wind.outside: 20m
weather:
  outside:
    rain: rain.outside
    temp: temp.garden
    wind: wind.outside
  windy: 3.2
wunderground:
  id: STATIONID
  password: password
  url: http://weatherstation.wunderground.com/weatherstation/updateweatherstation.php`

var ExampleConfig = Must(OpenReader(strings.NewReader(ExampleYaml)))
