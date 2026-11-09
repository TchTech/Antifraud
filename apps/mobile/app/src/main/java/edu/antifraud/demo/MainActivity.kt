package edu.antifraud.demo

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.util.UUID

// This transport is only for an isolated synthetic demo. It holds no credentials.
class DemoApi(private val baseUrl: String) {
    suspend fun request(path: String, body: JSONObject? = null): JSONObject = withContext(Dispatchers.IO) {
        val connection = URL(baseUrl.trimEnd('/') + path).openConnection() as HttpURLConnection
        connection.connectTimeout = 2500
        connection.readTimeout = 3000
        try {
            if (body != null) {
                connection.requestMethod = "POST"
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "application/json")
                connection.outputStream.use { it.write(body.toString().toByteArray(Charsets.UTF_8)) }
            }
            val status = connection.responseCode
            val stream = if (status in 200..299) connection.inputStream else connection.errorStream
            val text = stream?.bufferedReader()?.use { it.readText() } ?: "HTTP $status"
            check(status in 200..299) { "HTTP $status: $text" }
            JSONObject(text)
        } finally { connection.disconnect() }
    }
}

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { MaterialTheme { DemoScreen() } }
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun DemoScreen() {
    var baseUrl by remember { mutableStateOf("http://10.0.2.2:8080") }
    var account by remember { mutableStateOf("demo-legal") }
    var result by remember { mutableStateOf("Выберите синтетический сценарий. Назначение: sink-a.") }
    var operationId by remember { mutableStateOf("") }
    var confirmationId by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    fun runRequest(block: suspend () -> Unit) {
        busy = true
        scope.launch { try { block() } catch (error: Exception) { result = error.message ?: "Ошибка связи" } finally { busy = false } }
    }
    Surface(modifier = Modifier.fillMaxSize()) {
        Column(Modifier.padding(24.dp).verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Text("Сессионный антифрод", style = MaterialTheme.typography.headlineMedium)
            Text("Учебный стенд · только синтетика. Аттестация и подтверждение — тестовые подстановки. Реальные номера не вводятся.")
            OutlinedTextField(value = baseUrl, onValueChange = { baseUrl = it }, label = { Text("Адрес изолированного стенда") }, enabled = !busy)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                for (item in listOf("demo-legal", "demo-fraud")) {
                    FilterChip(selected = account == item, onClick = { account = item; confirmationId = "" }, label = { Text(item) }, enabled = !busy)
                }
            }
            Text("Аккаунт: $account\nОперация: включить переадресацию → sink-a")
            Button(enabled = !busy, onClick = {
                confirmationId = ""
                operationId = UUID.randomUUID().toString()
                runRequest {
                    val decision = DemoApi(baseUrl).request("/v1/evaluate", JSONObject()
                        .put("accountId", account).put("operationId", operationId).put("target", "sink-a")
                        .put("device", if (account == "demo-fraud") "new" else "known").put("attestation", "demo-valid"))
                    confirmationId = decision.optString("confirmationId")
                    result = "Решение: ${decision.getString("result")}\nПолитика: ${decision.getString("policyVersion")}\n${decision.getString("decisionId")}"
                }
            }) { Text("Проверить защищённое действие") }
            if (confirmationId.isNotEmpty()) {
                Text("Подтвердить конкретное изменение $account → sink-a? Это тестовая замена независимого подтверждения.")
                Button(enabled = !busy, onClick = {
                    runRequest {
                        val decision = DemoApi(baseUrl).request("/v1/confirm", JSONObject().put("accountId", account)
                            .put("operationId", operationId).put("confirmationId", confirmationId).put("proof", "demo-approved"))
                        confirmationId = ""
                        result = "Решение: ${decision.getString("result")}. Доставка команды выполняется отдельно."
                    }
                }) { Text("Тест: подтверждаю sink-a") }
            }
            Button(enabled = !busy, onClick = {
                runRequest {
                    val state = DemoApi(baseUrl).request("/v1/state/$account")
                    result = "Услуга на сервере: ${state.getString("target")}\nИзменений: ${state.getInt("changes")}\nРешений: ${state.getJSONArray("decisions").length()}\nАудит: ${state.getJSONArray("audit").length()} записей"
                }
            }) { Text("Проверить состояние услуги") }
            if (busy) LinearProgressIndicator()
            Text(result, style = MaterialTheme.typography.bodyLarge)
        }
    }
}
