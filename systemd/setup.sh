#!/bin/sh
# Script to setup gohome to run under the user systemd.

# Adjust this to the gohome services you're using.
SERVICES='api arduino automata bills camera currentcost datalogger earth
espeaker graphite heating irrigation jabber pubsub rfid rfxtrx sms systemd
twitter watchdog weather wunderground xpl'

# enable persistent user systemd
sudo loginctl enable-linger $USER

# copy service config
mkdir -p ~/.config/systemd/user
cp gohome@.service ~/.config/systemd/user
cp redis-ready.service ~/.config/systemd/user
 
# enable and start services
for SERVICE in $SERVICES; do
	echo "Enabling and starting $SERVICE"
	systemctl --user enable --now gohome@$SERVICE
done

echo "Done"
