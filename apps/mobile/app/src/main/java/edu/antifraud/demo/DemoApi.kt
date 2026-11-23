package edu.antifraud.demo

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

internal data class DemoDecision(
    val outcome: String,
    val version: String,
    val id: String,
    val confirmationId: String?,
)

internal data class DemoService(val target: String, val changes: Int, val decisions: Int, val auditEntries: Int)

// The two connections are the isolated emulator bridge and the local adb tunnel.
internal class DemoApi(private val connection: StandConnection) {
    suspend fun evaluate(scenario: DemoScenario, operationId: String): DemoDecision =
        request("/v1/evaluate", JSONObject()
            .put("accountId", scenario.account)
            .put("operationId", operationId)
            .put("target", TARGET)
            .put("device", scenario.device)
            .put("attestation", "demo-valid")).decision()

    suspend fun confirm(scenario: DemoScenario, operationId: String, confirmationId: String): DemoDecision =
        request("/v1/confirm", JSONObject()
            .put("accountId", scenario.account)
            .put("operationId", operationId)
            .put("confirmationId", confirmationId)
            .put("proof", "demo-approved")).decision()

    suspend fun service(scenario: DemoScenario): DemoService {
        val response = request("/v1/state/${scenario.account}")
        return DemoService(
            target = response.getString("target"),
            changes = response.getInt("changes"),
            decisions = response.historySize("decisions"),
            auditEntries = response.historySize("audit"),
        )
    }

    private fun JSONObject.historySize(key: String): Int =
        if (has(key) && isNull(key)) 0 else getJSONArray(key).length()

    private fun JSONObject.decision(): DemoDecision {
        val outcome = getString("result")
        return DemoDecision(
            outcome = outcome,
            version = getString("policyVersion"),
            id = getString("decisionId"),
            confirmationId = optString("confirmationId").takeIf { outcome == "step_up" && it.isNotBlank() },
        )
    }

    private suspend fun request(path: String, body: JSONObject? = null): JSONObject =
        withContext(Dispatchers.IO) {
            val http = URL(connection.url + path).openConnection() as HttpURLConnection
            http.connectTimeout = 2500
            http.readTimeout = 3000
            http.instanceFollowRedirects = false
            try {
                if (body != null) {
                    http.requestMethod = "POST"
                    http.doOutput = true
                    http.setRequestProperty("Content-Type", "application/json")
                    http.outputStream.use { it.write(body.toString().toByteArray(Charsets.UTF_8)) }
                }
                val status = http.responseCode
                check(status in 200..299) { "HTTP $status" }
                http.inputStream.bufferedReader(Charsets.UTF_8).use { JSONObject(it.readText()) }
            } finally {
                http.disconnect()
            }
        }
}
