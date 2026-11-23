package edu.antifraud.demo

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.core.view.WindowCompat
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.util.UUID

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.navigationBarColor = MobileColors.Surface.toArgb()
        WindowCompat.getInsetsController(window, window.decorView).apply {
            isAppearanceLightStatusBars = true
            isAppearanceLightNavigationBars = true
        }
        setContent {
            MobileTheme {
                DemoApp { isRibbon ->
                    window.statusBarColor = (if (isRibbon) MobileColors.Ice else MobileColors.Surface).toArgb()
                }
            }
        }
    }
}

internal const val TARGET = "sink-a"

internal enum class DemoScenario(val account: String, val title: String, val device: String) {
    LEGAL("demo-legal", "Замена SIM", "known"),
    FRAUD("demo-fraud", "Подозрительный вход", "new"),
}

internal enum class StandConnection(val title: String, val url: String) {
    EMULATOR("Эмулятор", "http://10.0.2.2:8080"),
    DEVICE("Устройство · adb reverse", "http://127.0.0.1:8080"),
}

internal enum class Page { HOME, EXPENSES, CATALOG, SUPPORT, CALLS, SERVICES, FORWARDING }

private data class PendingConfirmation(val operationId: String, val confirmationId: String)

@Composable
private fun DemoApp(onStatusBarChanged: (Boolean) -> Unit) {
    var page by rememberSaveable { mutableStateOf(Page.HOME) }
    var tab by rememberSaveable { mutableStateOf(Page.HOME) }
    var forwardingOrigin by rememberSaveable { mutableStateOf(Page.CALLS) }
    var connection by rememberSaveable { mutableStateOf(StandConnection.EMULATOR) }
    var scenario by rememberSaveable { mutableStateOf(DemoScenario.LEGAL) }
    var pending by remember { mutableStateOf<PendingConfirmation?>(null) }
    var decision by remember { mutableStateOf<DemoDecision?>(null) }
    var service by remember { mutableStateOf<DemoService?>(null) }
    var failure by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun clearResult() {
        pending = null
        decision = null
        service = null
        failure = null
    }

    fun request(onFailure: String, action: suspend (DemoApi) -> Unit) {
        if (busy) return
        busy = true
        failure = null
        val api = DemoApi(connection)
        scope.launch {
            try {
                action(api)
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (_: Exception) {
                failure = onFailure
            } finally {
                busy = false
            }
        }
    }

    fun selectTab(destination: Page) {
        tab = destination
        page = destination
    }

    fun openForwarding() {
        forwardingOrigin = page
        page = Page.FORWARDING
    }

    fun goBack() {
        when (page) {
            Page.FORWARDING -> page = forwardingOrigin
            Page.SERVICES -> page = tab
            else -> selectTab(Page.HOME)
        }
    }

    BackHandler(enabled = page != Page.HOME) { goBack() }
    SideEffect {
        onStatusBarChanged(page in listOf(Page.HOME, Page.CATALOG, Page.SUPPORT, Page.CALLS))
    }
    Scaffold(
        modifier = Modifier.fillMaxSize(),
        containerColor = MobileColors.Surface,
        bottomBar = { MobileNavigation(tab, ::selectTab) },
    ) { insets ->
        Box(Modifier.fillMaxSize().padding(insets)) {
            key(page) {
                when (page) {
                    Page.HOME -> HomePage(
                        onExpenses = { selectTab(Page.EXPENSES) },
                        onServices = { page = Page.SERVICES },
                        onForwarding = ::openForwarding,
                        onSupport = { selectTab(Page.SUPPORT) },
                        onSearch = { selectTab(Page.CATALOG) },
                    )
                    Page.EXPENSES -> ExpensesPage(::goBack)
                    Page.CATALOG -> CatalogPage(
                        onServices = { page = Page.SERVICES },
                        onForwarding = ::openForwarding,
                        onSupport = { selectTab(Page.SUPPORT) },
                    )
                    Page.SUPPORT -> SupportPage(connection, busy) { selected ->
                        if (selected != connection) {
                            connection = selected
                            clearResult()
                        }
                    }
                    Page.CALLS -> CallsPage(::openForwarding)
                    Page.SERVICES -> ServicesPage(::goBack, ::openForwarding)
                    Page.FORWARDING -> ForwardingPage(
                        scenario = scenario,
                        onBack = ::goBack,
                        onScenario = { selected ->
                            if (selected != scenario) {
                                scenario = selected
                                clearResult()
                            }
                        },
                        decision = decision,
                        awaitingConfirmation = pending != null,
                        service = service,
                        failure = failure,
                        busy = busy,
                        onEvaluate = {
                            clearResult()
                            val selected = scenario
                            val operationId = UUID.randomUUID().toString()
                            request("Не удалось проверить действие. Проверьте подключение к стенду.") { api ->
                                val result = api.evaluate(selected, operationId)
                                decision = result
                                pending = result.confirmationId?.let { PendingConfirmation(operationId, it) }
                            }
                        },
                        onConfirm = {
                            val challenge = pending
                            if (challenge != null && !busy) {
                                pending = null
                                val selected = scenario
                                service = null
                                request("Не удалось подтвердить действие. Проверьте состояние услуги перед новой попыткой.") { api ->
                                    decision = api.confirm(selected, challenge.operationId, challenge.confirmationId)
                                }
                            }
                        },
                        onCancelConfirmation = { pending = null },
                        onRefresh = {
                            val selected = scenario
                            request("Не удалось получить состояние услуги. Проверьте подключение к стенду.") { api ->
                                service = api.service(selected)
                            }
                        },
                    )
                }
            }
        }
    }
}
