package edu.antifraud.demo

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountCircle
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp

private const val ALL_CALLS = "Все вызовы"
private const val TRAINING_NUMBER_LABEL = "Учебный номер"

@Composable
internal fun ForwardingPage(
    scenario: DemoScenario,
    onBack: () -> Unit,
    outcome: String?,
    awaitingConfirmation: Boolean,
    service: DemoService?,
    failure: String?,
    busy: Boolean,
    onEvaluate: () -> Unit,
    onConfirm: () -> Unit,
    onCancelConfirmation: () -> Unit,
    onRefresh: () -> Unit,
) {
    var adding by remember { mutableStateOf(false) }
    val active = service != null && service.target != "disabled"
    Column(Modifier.fillMaxSize()) {
        Box {
            PlainHeader("Мой МТС", "Учебный профиль", onBack)
            Icon(
                Icons.Filled.AccountCircle,
                contentDescription = null,
                tint = MobileColors.Muted.copy(alpha = 0.62f),
                modifier = Modifier.align(Alignment.CenterEnd).padding(end = 16.dp).size(32.dp),
            )
        }
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState())) {
            ScreenContent {
                Text("Переадресация вызовов", style = MaterialTheme.typography.headlineMedium)

                if (active) {
                    InfoCard(modifier = Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Polite }) {
                        Text(ALL_CALLS, style = MaterialTheme.typography.titleMedium)
                        Text(TRAINING_NUMBER_LABEL, style = MaterialTheme.typography.bodyLarge)
                        Text("Подключена", color = MobileColors.Green, style = MaterialTheme.typography.bodyMedium)
                    }
                } else {
                    InfoCard(modifier = Modifier.fillMaxWidth(), color = MobileColors.Neutral) {
                        Text("Перенаправляйте входящие звонки на другой номер или на голосовую почту.")
                    }
                    ForwardingBullet("Безусловная переадресация", "переводит все вызовы")
                    ForwardingBullet("Условная переадресация", "срабатывает, когда вы заняты, недоступны или не отвечаете")
                    ForwardingBullet("Голосовая почта", "сохраняет сообщения, которые можно прослушать позже")
                }

                if (outcome == "hold") {
                    InfoCard(modifier = Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Polite }, color = MobileColors.Rose) {
                        Text("Переадресация не добавлена", style = MaterialTheme.typography.titleMedium)
                        Text("Изменение временно недоступно. Попробуйте позже.")
                    }
                }
                if (failure != null || (outcome == "allow" && !active)) {
                    InfoCard(modifier = Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Polite }, color = MobileColors.Neutral) {
                        Text(failure ?: "Настройка ещё применяется.")
                        TextButton(onClick = onRefresh, enabled = !busy) { Text("Обновить") }
                    }
                }
            }
        }
        if (busy) {
            LinearProgressIndicator(Modifier.fillMaxWidth())
        }
        if (!active) {
            Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 12.dp)) {
                ForwardingButton("Добавить переадресацию", !busy && outcome != "allow") { adding = true }
            }
        }
    }

    if (adding) {
        AddForwardingSheet(onClose = { adding = false }, onAdd = {
            adding = false
            onEvaluate()
        })
    }
    if (awaitingConfirmation) {
        AlertDialog(
            onDismissRequest = { if (!busy) onCancelConfirmation() },
            containerColor = MobileColors.Surface,
            title = { Text("Подтвердите переадресацию") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Все вызовы будут перенаправляться на учебный номер.")
                    Text("Аккаунт ${scenario.account}")
                    Text("Подтверждение в учебном профиле", color = MobileColors.Muted)
                }
            },
            confirmButton = { TextButton(enabled = !busy, onClick = onConfirm) { Text("Подтвердить") } },
            dismissButton = { TextButton(enabled = !busy, onClick = onCancelConfirmation) { Text("Отмена") } },
        )
    }
}

@Composable
private fun ForwardingBullet(title: String, description: String) {
    Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.Top) {
        Box(Modifier.padding(top = 8.dp).size(7.dp).background(MobileColors.Red, CircleShape))
        Text("$title — $description", style = MaterialTheme.typography.bodyLarge)
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun AddForwardingSheet(onClose: () -> Unit, onAdd: () -> Unit) {
    var selected by remember { mutableStateOf(false) }
    var expanded by remember { mutableStateOf(false) }
    ModalBottomSheet(
        onDismissRequest = onClose,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = MobileColors.Surface,
        shape = RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp),
        tonalElevation = 0.dp,
        dragHandle = null,
    ) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState())) {
            ScreenContent {
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Text("Добавить переадресацию", style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
                    IconButton(onClick = onClose) {
                        Box(Modifier.size(36.dp).background(MobileColors.Neutral, CircleShape), contentAlignment = Alignment.Center) {
                            Icon(Icons.Filled.Close, "Закрыть")
                        }
                    }
                }
                Column(Modifier.fillMaxWidth()) {
                    OutlinedButton(
                        onClick = { expanded = !expanded },
                        modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp),
                        shape = RoundedCornerShape(8.dp),
                        border = BorderStroke(if (expanded) 2.dp else 1.dp, if (expanded) MobileColors.Blue else MobileColors.Muted),
                    ) {
                        Text(
                            if (selected) ALL_CALLS else "Выберите условие",
                            color = if (selected) MobileColors.Ink else MobileColors.Muted,
                            modifier = Modifier.weight(1f),
                        )
                        Icon(
                            if (expanded) Icons.Filled.KeyboardArrowUp else Icons.Filled.KeyboardArrowDown,
                            contentDescription = null,
                            tint = MobileColors.Muted,
                        )
                    }
                    if (expanded) {
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            color = MobileColors.Surface,
                            shape = RoundedCornerShape(8.dp),
                            shadowElevation = 6.dp,
                        ) {
                            Column {
                                DropdownMenuItem(text = { Text(ALL_CALLS) }, onClick = {
                                    selected = true
                                    expanded = false
                                })
                                for (condition in listOf("Если занято", "Если недоступен", "Если не отвечаю")) {
                                    DropdownMenuItem(text = { Text(condition) }, enabled = false, onClick = {})
                                }
                            }
                        }
                    }
                }
                if (selected) {
                    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text("Переадресовать на номер", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(
                            value = TRAINING_NUMBER_LABEL,
                            onValueChange = {},
                            readOnly = true,
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                }
                InfoCard(modifier = Modifier.fillMaxWidth(), color = MobileColors.Neutral) {
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        Icon(Icons.Filled.Info, contentDescription = null, tint = MobileColors.Ocean, modifier = Modifier.size(20.dp))
                        Text("Для переадресации на голосовую почту потребуется отдельная услуга.", style = MaterialTheme.typography.bodyMedium)
                    }
                }
                ForwardingButton("Добавить", selected, onAdd)
            }
        }
    }
}

@Composable
private fun ForwardingButton(label: String, enabled: Boolean, onClick: () -> Unit) {
    Button(
        modifier = Modifier.fillMaxWidth().heightIn(min = 52.dp),
        shape = RoundedCornerShape(8.dp),
        colors = ButtonDefaults.buttonColors(containerColor = MobileColors.ActionRed),
        enabled = enabled,
        onClick = onClick,
    ) {
        Text(label, style = MaterialTheme.typography.titleMedium)
    }
}
