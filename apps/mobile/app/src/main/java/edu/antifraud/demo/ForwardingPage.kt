package edu.antifraud.demo

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp

@Composable
@OptIn(ExperimentalMaterial3Api::class)
internal fun ForwardingPage(
    scenario: DemoScenario,
    onBack: () -> Unit,
    onScenario: (DemoScenario) -> Unit,
    decision: DemoDecision?,
    awaitingConfirmation: Boolean,
    service: DemoService?,
    failure: String?,
    busy: Boolean,
    onEvaluate: () -> Unit,
    onConfirm: () -> Unit,
    onCancelConfirmation: () -> Unit,
    onRefresh: () -> Unit,
) {
    ScrollPage {
        PlainHeader("Переадресация", "Учебный стенд", onBack)
        Divider()
        ScreenContent {
            InfoCard(modifier = Modifier.fillMaxWidth(), color = MobileColors.Ice) {
                ServiceRow("Переадресация вызовов", "Включить для $TARGET", Icons.Filled.Call, MobileColors.Blue, MobileColors.Surface)
            }
            SectionTitle("Сценарий")
            for (option in DemoScenario.entries) {
                FilterChip(
                    selected = option == scenario,
                    enabled = !busy,
                    onClick = { onScenario(option) },
                    label = { Text(option.title) },
                )
            }
            Text(scenario.account, color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
            Button(modifier = Modifier.fillMaxWidth(), enabled = !busy, onClick = onEvaluate) {
                Text("Проверить защищённое действие")
            }
            if (busy) {
                LinearProgressIndicator(Modifier.fillMaxWidth())
            }
            if (decision != null) {
                DecisionCard(decision)
            }
            if (failure != null) {
                InfoCard(modifier = Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Polite }, color = MobileColors.Rose) {
                    Text(failure, style = MaterialTheme.typography.bodyLarge)
                }
            }
            OutlinedButton(modifier = Modifier.fillMaxWidth(), enabled = !busy, onClick = onRefresh) {
                Text("Проверить состояние услуги")
            }
            if (service != null) {
                InfoCard(modifier = Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Polite }, color = MobileColors.Neutral) {
                    Text("Состояние услуги", style = MaterialTheme.typography.titleMedium)
                    Text(
                        if (service.target == "disabled") "Переадресация выключена" else "Назначение: ${service.target}",
                        style = MaterialTheme.typography.bodyLarge,
                    )
                    Text("Изменений: ${service.changes}", style = MaterialTheme.typography.bodyMedium)
                    Text("Решений: ${service.decisions} · Аудит: ${service.auditEntries}", style = MaterialTheme.typography.bodyMedium)
                }
            }
        }
    }

    if (awaitingConfirmation) {
        AlertDialog(
            onDismissRequest = { if (!busy) onCancelConfirmation() },
            containerColor = MobileColors.Surface,
            icon = { IconTile(Icons.Filled.Lock, MobileColors.Ice, MobileColors.Ocean) },
            title = { Text("Подтвердите переадресацию") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Аккаунт: ${scenario.account}")
                    Text("Действие: включить переадресацию")
                    Text("Назначение: $TARGET")
                    Text("Это тестовое подтверждение учебного стенда.", color = MobileColors.Muted)
                }
            },
            confirmButton = {
                Button(enabled = !busy, onClick = onConfirm) { Text("Подтвердить для $TARGET") }
            },
            dismissButton = {
                TextButton(enabled = !busy, onClick = onCancelConfirmation) { Text("Отмена") }
            },
        )
    }
}

private enum class Outcome(val title: String, val description: String) {
    HOLD("Действие приостановлено", "Для этой операции переадресация не будет включена."),
    STEP_UP("Нужно подтверждение", "Проверка требует подтверждения перед включением переадресации."),
    ALLOW("Действие разрешено", "Команда отправлена на исполнение. Фактический результат доступен в состоянии услуги."),
    UNKNOWN("Неизвестное решение", "Посмотрите состояние услуги и проверьте версию стенда."),
}

@Composable
private fun DecisionCard(decision: DemoDecision) {
    val outcome = when (decision.outcome) {
        "hold" -> Outcome.HOLD
        "step_up" -> Outcome.STEP_UP
        "allow" -> Outcome.ALLOW
        else -> Outcome.UNKNOWN
    }
    val background = when (outcome) {
        Outcome.HOLD, Outcome.UNKNOWN -> MobileColors.Rose
        Outcome.STEP_UP -> MobileColors.Ice
        Outcome.ALLOW -> MobileColors.Mint
    }
    val foreground = when (outcome) {
        Outcome.HOLD, Outcome.UNKNOWN -> MobileColors.ActionRed
        Outcome.STEP_UP -> MobileColors.Ocean
        Outcome.ALLOW -> MobileColors.Green
    }
    InfoCard(
        modifier = Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Polite },
        color = background,
    ) {
        Text(outcome.title, color = foreground, style = MaterialTheme.typography.titleMedium)
        Text(outcome.description, style = MaterialTheme.typography.bodyLarge)
        Text("${decision.outcome} · ${decision.version}", color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
        Text(decision.id, color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
    }
}
