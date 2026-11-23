package edu.antifraud.demo

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountBox
import androidx.compose.material.icons.filled.ArrowForward
import androidx.compose.material.icons.filled.Build
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.List
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Phone
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

@Composable
internal fun HomePage(
    onExpenses: () -> Unit,
    onServices: () -> Unit,
    onForwarding: () -> Unit,
    onSupport: () -> Unit,
    onSearch: () -> Unit,
) {
    ScrollPage {
        RibbonHeader {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconTile(Icons.Filled.Person, MobileColors.Blue)
                Column(Modifier.weight(1f)) {
                    Text("Основной аккаунт", color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
                    Text("Учебный профиль", style = MaterialTheme.typography.titleMedium)
                }
                Pill("Демо", MobileColors.Violet, MobileColors.Lavender)
            }
            Surface(onClick = onSearch, shape = RoundedCornerShape(12.dp), color = MobileColors.Surface) {
                Row(
                    Modifier.fillMaxWidth().heightIn(min = 48.dp).padding(horizontal = 12.dp, vertical = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(Icons.Filled.Search, contentDescription = null, tint = MobileColors.Muted)
                    Text("Найти услугу", style = MaterialTheme.typography.bodyLarge, color = MobileColors.Muted)
                }
            }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                HomeShortcut("Мои услуги", Icons.Filled.List, onServices, Modifier.weight(1f), MobileColors.Ice, MobileColors.Ocean)
                HomeShortcut("Переадресация", Icons.Filled.Call, onForwarding, Modifier.weight(1f), MobileColors.Rose, MobileColors.Red)
                HomeShortcut("Стенд", Icons.Filled.Settings, onSupport, Modifier.weight(1f), MobileColors.Lavender, MobileColors.Violet)
            }
        }
        Box(
            Modifier.fillMaxWidth().clip(RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp))
                .background(MobileColors.Backdrop),
        ) {
            ScreenContent {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("153,29 ₽", style = MaterialTheme.typography.headlineLarge, modifier = Modifier.weight(1f))
                    Button(onClick = onExpenses) { Text("Расходы") }
                }
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    InfoCard(Modifier.weight(1f), onClick = onExpenses) {
                        Text("Расходы в августе", style = MaterialTheme.typography.labelLarge)
                        Text("151 ₽", style = MaterialTheme.typography.titleMedium)
                        UsageTrack(.83f)
                    }
                    InfoCard(Modifier.weight(1f), onClick = onServices) {
                        Text("Подключённые услуги", style = MaterialTheme.typography.labelLarge)
                        Text("Тариф и звонки", color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
                    }
                }
                TariffCard()
                SectionTitle("Управление услугами")
                InfoCard(onClick = onForwarding) {
                    ServiceRow("Переадресация вызовов", "Подключение с проверкой безопасности", Icons.Filled.Call)
                }
            }
        }
    }
}

@Composable
private fun HomeShortcut(
    title: String,
    icon: ImageVector,
    onClick: () -> Unit,
    modifier: Modifier,
    background: Color,
    tint: Color,
) {
    InfoCard(modifier, color = background, onClick = onClick) {
        IconTile(icon, Color.White, tint)
        Text(title, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
internal fun ExpensesPage(onBack: () -> Unit) {
    ScrollPage {
        PlainHeader("Контроль расходов", "Учебный профиль", onBack)
        Divider()
        ScreenContent {
            SectionTitle("История расходов")
            Text("Август", color = MobileColors.Ocean, style = MaterialTheme.typography.labelLarge)
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.weight(1f)) { UsageTrack(1f) }
                Text("151 ₽", style = MaterialTheme.typography.titleMedium)
            }
            Text("14 августа", color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
            ServiceRow("Полный безлимит", "Интернет", Icons.Filled.Phone, trailing = "−1 ₽")
            ServiceRow("Тариф «Персональный»", "Абонентская плата", Icons.Filled.DateRange, trailing = "−150 ₽")
        }
    }
}

@Composable
internal fun ServicesPage(onBack: () -> Unit, onForwarding: () -> Unit) {
    ScrollPage {
        PlainHeader("Мои услуги", "Учебный профиль", onBack)
        Divider()
        ScreenContent {
            SectionTitle("Тарифы и подписки")
            TariffCard()
            SectionTitle("Подключённые услуги связи")
            ServiceRow("Полный безлимит", "Доступ к интернету", Icons.Filled.Phone, trailing = "1 ₽/мес")
            ServiceRow("За границей", "Списание при использовании в роуминге", Icons.Filled.LocationOn, trailing = "550 ₽/день")
            SectionTitle("Управление звонками")
            ServiceRow("Переадресация вызовов", "Включение с дополнительной проверкой", Icons.Filled.Call, onClick = onForwarding)
        }
    }
}

@Composable
private fun TariffCard() {
    InfoCard(modifier = Modifier.fillMaxWidth()) {
        Text("Персональный", style = MaterialTheme.typography.titleMedium)
        Text("150 ₽/мес", style = MaterialTheme.typography.bodyMedium, color = MobileColors.Muted)
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            TariffUsage("∞ ГБ", "Интернет", .95f, Modifier.weight(1f))
            TariffUsage("400 мин", "Звонки", .86f, Modifier.weight(1f))
            TariffUsage("100 SMS", "Сообщения", .64f, Modifier.weight(1f))
        }
    }
}

@Composable
private fun TariffUsage(value: String, label: String, fraction: Float, modifier: Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(value, style = MaterialTheme.typography.labelLarge)
        UsageTrack(fraction)
        Text(label, style = MaterialTheme.typography.bodyMedium, color = MobileColors.Muted)
    }
}

@Composable
private fun UsageTrack(fraction: Float) {
    Box(Modifier.fillMaxWidth().height(4.dp).clip(CircleShape).background(MobileColors.Ice)) {
        Box(Modifier.fillMaxWidth(fraction).height(4.dp).clip(CircleShape).background(MobileColors.Blue))
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
internal fun CatalogPage(onServices: () -> Unit, onForwarding: () -> Unit, onSupport: () -> Unit) {
    var query by rememberSaveable { mutableStateOf("") }
    val offerings = listOf(
        CatalogItem("Тарифы", "Условия и пакеты по тарифу", Icons.Filled.AccountBox, onServices),
        CatalogItem("Услуги", "Интернет и управление звонками", Icons.Filled.List, onServices),
        CatalogItem("Переадресация", "Управление входящими вызовами", Icons.Filled.ArrowForward, onForwarding),
    ).filter { it.title.contains(query.trim(), ignoreCase = true) || it.subtitle.contains(query.trim(), ignoreCase = true) }
    ScrollPage {
        RibbonHeader { RibbonTitle("Каталог", "Учебный профиль") }
        ScreenContent {
            OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Поиск услуг") },
                leadingIcon = { Icon(Icons.Filled.Search, contentDescription = null) },
                singleLine = true,
            )
            if (query.isBlank()) {
                InfoCard(modifier = Modifier.fillMaxWidth(), color = MobileColors.Lavender, onClick = onServices) {
                    Text("Услуги для вашего номера", style = MaterialTheme.typography.titleMedium)
                    Text("Посмотрите тариф и подключённые услуги", color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
                }
            }
            SectionTitle("Мобильная связь")
            for (item in offerings) {
                ServiceRow(item.title, item.subtitle, item.icon, MobileColors.Blue, Color.White, onClick = item.action)
            }
            if (offerings.isEmpty()) {
                Text("Услуги не найдены. Попробуйте другой запрос.", style = MaterialTheme.typography.bodyLarge)
            }
            if (query.isBlank()) {
                SectionTitle("Поддержка")
                ServiceRow("Настройки стенда", "Подключение учебного приложения", Icons.Filled.Build, MobileColors.Violet, Color.White, onClick = onSupport)
            }
        }
    }
}

private data class CatalogItem(
    val title: String,
    val subtitle: String,
    val icon: ImageVector,
    val action: () -> Unit,
)

@Composable
@OptIn(ExperimentalMaterial3Api::class)
internal fun SupportPage(connection: StandConnection, busy: Boolean, onConnection: (StandConnection) -> Unit) {
    ScrollPage {
        RibbonHeader { RibbonTitle("Поддержка", "Учебный профиль") }
        ScreenContent {
            InfoCard(modifier = Modifier.fillMaxWidth(), color = MobileColors.Ice) {
                Text("Подключение к стенду", style = MaterialTheme.typography.titleMedium)
                Text("Поднимите шлюз и сборщик сигналов, затем выберите способ подключения.", style = MaterialTheme.typography.bodyLarge)
            }
            for (option in StandConnection.entries) {
                FilterChip(
                    selected = connection == option,
                    onClick = { onConnection(option) },
                    enabled = !busy,
                    label = { Text(option.title) },
                )
            }
            Text(connection.url, style = MaterialTheme.typography.bodyMedium, color = MobileColors.Muted)
            if (connection == StandConnection.DEVICE) {
                Text("На компьютере выполните adb reverse tcp:8080 tcp:8080.", style = MaterialTheme.typography.bodyMedium)
            }
            Divider()
            SectionTitle("О приложении")
            ServiceRow("Учебный стенд", "Все профили, сигналы и назначения синтетические.", Icons.Filled.Info)
            ServiceRow("Проверка переадресации", "Подтверждение и аттестация используются в тестовом режиме.", Icons.Filled.Lock)
        }
    }
}

@Composable
internal fun CallsPage(onForwarding: () -> Unit) {
    ScrollPage {
        RibbonHeader { RibbonTitle("Звонки", "Учебный профиль") }
        ScreenContent {
            SectionTitle("Управление звонками")
            ServiceRow(
                "Переадресация вызовов",
                "Проверьте подключение и состояние услуги",
                Icons.Filled.Call,
                MobileColors.Blue,
                Color.White,
                onClick = onForwarding,
            )
        }
    }
}
