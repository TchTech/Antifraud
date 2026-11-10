# Android demo

Kotlin 1.9.10, Compose compiler 1.5.3, AGP 8.1.1, Gradle wrapper 8.2.1, JDK 17, Android SDK 34. Сборка: ./gradlew assembleDebug (Windows: .\gradlew.bat assembleDebug), установка: adb install -r app/build/outputs/apk/debug/app-debug.apk.

Compose состояние держи явно; сетевые вызовы — Dispatchers.IO, исключения показывай в UI. Предпочитай простые классы, понятные Java-разработчику; модели и таймауты явные. Сохраняй disabled-состояние кнопок во время запроса. Экран подтверждения показывает аккаунт, действие и конкретный sink. Секретов и реальной телеметрии здесь нет.

Эмулятор использует http://10.0.2.2:8080. Для adb reverse tcp:8080 tcp:8080 установи http://127.0.0.1:8080. HTTP и тестовые значения предназначены для изолированного демо; production TLS, Play Integrity и реальные подтверждения — отдельные задачи OpenSpec. Физическое устройство пока не проверено.
