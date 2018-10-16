# qrget

Simple PC/Laptop phone/tablet download tool using QR Codes and HTTP protocol.
No external tools on your mobile device are needed, no setup of Bluetooth or
missing NFC (I am watching you Xiaomi) or USB cables required.

**Warning:** `qrcode` is in development mode, not yet user friendly, not
working elsewhere than on Linux (rely on `sysfs`), can't handle command line
argument and do not have GUI. Please be patient or send pull requests (it is
BSD licensed)

## How it works

```
      ._________________.                 FILE               .__________________.   
      |.---------------.|   ---------------------------->    |.----------------.|   
      ||     .____.    ||             .________.             ||         .:'    ||             
      ||     | QR |    ||   FILE      |.------.|             ||     __ :'__    ||             
      ||     |code|    ||   ---->     || AN   ||             ||  .'`__`-'__``. ||             
      ||     |____|    ||             || DRO  ||             || :__________.-' ||             
      ||_______________||             || ID   ||             || :_________:    ||             
      /.-.-.-.-.-.-.-.-.\             ||      ||             ||  :_________`-; ||             
     /.-.-.-.-.-.-.-.-.-.\            ||______||             jgs  `.__.-.__.'  ||
    /.-.-.-.-.-.-.-.-.-.-.\           .--------.             ||________________||
   /______/__________\___o_\ DrS        Phone +              ||      |   |     ||
   \_______________________/            Camera               .------------------.
         Desktop/Laptop                                         Tables + camera          
                                           ^
                                           |
              ^                    \       |      /                   ^
              |                     \      |     /                    |
              |                      \     |    /                     |
              |                     __\____|___/__                    |
              .--------------      /__o__o__o__o__\     --------------.
                                   \______________/
                                          WiFi
```

 1. Your phone/tablet MUST be connected to same WiFi as Laptop
 2. Your phone/tablet MUST have camera
 3. Your phone/tablet MUST have QR code Reader
 4. You run application `go run main.go`
 5. You open QR Code `download.png`
 6. Use QR Code Reader on your mobile device to download the file

## How it works really

It is stupid simple. `qrcode` generates and encodes URL to download the file
with WLAN address of main computer and run HTTP server serving that file. QR
Code reader decodes URL and open web browser to download the file.

## Credits
 * [ASCII Art Laptop](http://ascii.co.uk/art/laptop)
 * [ASCII Art Apple logo](https://www.asciiart.eu/computers/apple)