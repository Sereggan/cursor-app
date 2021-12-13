# cursor-app
Простое приложение, позволяющее отображать курсоры всех пользователей у которых открыта вкладка  
## Как запустить:  
 * make
 * Запустить main.go :) 
 * go run .
    
## Проект на 3 файла:  
 * main. go - запускает сервер  
 * client.go - создаёт 2 горутины на каждый коннекшн, в одной читает с вебсокета и добавляет в hub, в другой читаем с hub и отправляет по вебсокету браузеру
 * hub.go - Организует коммуникацию между горутинами  

В качестве фреймворка использовал gorilla/websocket  
https://github.com/gorilla/websocket  
взял за основу пример  
https://github.com/gorilla/websocket/tree/master/examples/chat  
