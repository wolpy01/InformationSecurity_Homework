# InformationSecurity_Homework_1

### Запуск прокси-сервера с https в Docker с хранением запросов в MongoDB
- Запустить `sudo docker-compose up`
- Собрать бинарный файл Go `make -B build`
- Запустить `./build/proxy/out`
- Скопировать сертификаты `sudo cp ~/.mitm/ca-cert.pem /usr/local/share/ca-certificates/ca-cert.crt`
- Обновить сертификаты `sudo update-ca-certificates`

### Тестовые запросы
- `curl -x http://127.0.0.1:8080 https://mail.ru`