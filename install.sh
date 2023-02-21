#!/bin/sh

echo "Installing..."

sudo apt install -y curl

sudo curl -# -L "https://raw.githubusercontent.com/Konixy/create-app/master/bin/create-app" -o /usr/bin/create-app

sudo chmod a+x /usr/bin/create-app

echo "Installation complete!"