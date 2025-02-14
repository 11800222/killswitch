# mods
- adds -w mode, which does the following:
  - loop for a vpn interface detected
  - once detected, write pass rule for the vpn **and** local to a tmp file
  - enable the rules as predefined anchor




# build 
```shell
sudo make
```


```shell
go mod tidy

go mod vendor

make
```


# deploy
```shell
chmod +x killswitch

sudo chown root:wheel killswitch

sudo chmod 744 killswitch
```