package edu.antifraud.demo

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountBox
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.Typography
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

internal object MobileColors {
    val Red = Color(0xFFFF0032)
    val ActionRed = Color(0xFFD6002B)
    val Ink = Color(0xFF1B2026)
    val Muted = Color(0xFF575F69)
    val Neutral = Color(0xFFF1F3F6)
    val Line = Color(0xFFE5E8ED)
    val Surface = Color(0xFFFFFFFF)
    val Backdrop = Color(0xFFF6F7FA)
    val Ocean = Color(0xFF0066B8)
    val Blue = Color(0xFF007BD4)
    val Ice = Color(0xFFDAECFF)
    val Lavender = Color(0xFFEAE5FC)
    val Violet = Color(0xFF653C9E)
    val Mint = Color(0xFFE2F5ED)
    val Green = Color(0xFF12664D)
    val Rose = Color(0xFFFFE8ED)
}

internal val CardShape = RoundedCornerShape(22.dp)
private val IconShape = RoundedCornerShape(13.dp)

@Composable
internal fun MobileTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = lightColorScheme(
            primary = MobileColors.ActionRed,
            onPrimary = Color.White,
            primaryContainer = MobileColors.Rose,
            onPrimaryContainer = MobileColors.Ink,
            background = MobileColors.Backdrop,
            surface = MobileColors.Surface,
            onSurface = MobileColors.Ink,
            onSurfaceVariant = MobileColors.Muted,
            outline = MobileColors.Muted,
            outlineVariant = MobileColors.Line,
        ),
        typography = Typography(
            headlineLarge = TextStyle(fontFamily = FontFamily.SansSerif, fontWeight = FontWeight.Bold, fontSize = 32.sp, lineHeight = 36.sp),
            headlineMedium = TextStyle(fontFamily = FontFamily.SansSerif, fontWeight = FontWeight.Bold, fontSize = 24.sp, lineHeight = 30.sp),
            titleMedium = TextStyle(fontFamily = FontFamily.SansSerif, fontWeight = FontWeight.Bold, fontSize = 17.sp, lineHeight = 22.sp),
            bodyLarge = TextStyle(fontFamily = FontFamily.SansSerif, fontWeight = FontWeight.Normal, fontSize = 15.sp, lineHeight = 21.sp),
            bodyMedium = TextStyle(fontFamily = FontFamily.SansSerif, fontWeight = FontWeight.Normal, fontSize = 13.sp, lineHeight = 18.sp),
            labelLarge = TextStyle(fontFamily = FontFamily.SansSerif, fontWeight = FontWeight.SemiBold, fontSize = 14.sp, lineHeight = 18.sp),
            labelSmall = TextStyle(fontFamily = FontFamily.SansSerif, fontWeight = FontWeight.Medium, fontSize = 10.sp, lineHeight = 12.sp),
        ),
        content = content,
    )
}

private data class Tab(val page: Page, val label: String, val icon: ImageVector)

private val tabs = listOf(
    Tab(Page.HOME, "Мой МТС", Icons.Filled.Home),
    Tab(Page.EXPENSES, "Деньги", Icons.Filled.AccountBox),
    Tab(Page.CATALOG, "Каталог", Icons.Filled.Menu),
    Tab(Page.SUPPORT, "Поддержка", Icons.Filled.Email),
    Tab(Page.CALLS, "Звонки", Icons.Filled.Call),
)

@Composable
internal fun MobileNavigation(selected: Page, onSelect: (Page) -> Unit) {
    Column {
        Divider()
        NavigationBar(
            containerColor = MobileColors.Surface,
            tonalElevation = 0.dp,
        ) {
            for (tab in tabs) {
                val active = tab.page == selected
                NavigationBarItem(
                    selected = active,
                    onClick = { onSelect(tab.page) },
                    icon = { Icon(tab.icon, contentDescription = null, modifier = Modifier.size(23.dp)) },
                    label = {
                        Text(
                            text = tab.label,
                            style = MaterialTheme.typography.labelSmall,
                            textAlign = TextAlign.Center,
                        )
                    },
                    colors = NavigationBarItemDefaults.colors(
                        selectedIconColor = MobileColors.Red,
                        selectedTextColor = MobileColors.ActionRed,
                        indicatorColor = MobileColors.Rose,
                        unselectedIconColor = MobileColors.Muted,
                        unselectedTextColor = MobileColors.Muted,
                    ),
                )
            }
        }
    }
}

@Composable
internal fun Divider() {
    Box(Modifier.fillMaxWidth().height(1.dp).background(MobileColors.Line))
}

@Composable
internal fun ScrollPage(content: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()),
        content = content,
    )
}

@Composable
internal fun ScreenContent(content: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier.fillMaxWidth().padding(start = 16.dp, end = 16.dp, top = 16.dp, bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
        content = content,
    )
}

@Composable
internal fun RibbonHeader(content: @Composable ColumnScope.() -> Unit) {
    Box(Modifier.fillMaxWidth().background(MobileColors.Ice)) {
        Canvas(Modifier.matchParentSize()) {
            val ribbon = Brush.horizontalGradient(
                0f to Color(0xFFB7D8F7),
                .22f to Color(0xFFF5FBFF),
                .41f to Color(0xFF69B0E6),
                .52f to Color(0xFF14699D),
                .65f to Color(0xFFB1DDF9),
                .78f to Color(0xFFEAF7FF),
                1f to Color(0xFFADD2F4),
                startX = size.width * .34f,
                endX = size.width * 1.15f,
            )
            rotate(20f, Offset(size.width * .7f, size.height / 2)) {
                drawRect(
                    ribbon,
                    topLeft = Offset(size.width * .24f, -size.height),
                    size = Size(size.width * .95f, size.height * 3),
                )
            }
            drawRect(Brush.horizontalGradient(listOf(MobileColors.Ice, MobileColors.Ice.copy(alpha = 0f))))
        }
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            content = content,
        )
    }
}

@Composable
internal fun RibbonTitle(title: String, subtitle: String) {
    Text(title, style = MaterialTheme.typography.headlineMedium)
    Text(subtitle, color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
}

@Composable
internal fun PlainHeader(title: String, subtitle: String, onBack: () -> Unit) {
    Box(Modifier.fillMaxWidth().heightIn(min = 62.dp), contentAlignment = Alignment.Center) {
        IconButton(onClick = onBack, modifier = Modifier.align(Alignment.CenterStart)) {
            Icon(Icons.Filled.ArrowBack, "Назад")
        }
        Column(
            Modifier.padding(horizontal = 52.dp, vertical = 8.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(title, style = MaterialTheme.typography.titleMedium, textAlign = TextAlign.Center)
            Text(subtitle, color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium, textAlign = TextAlign.Center)
        }
    }
}

@Composable
internal fun SectionTitle(title: String) {
    Text(title, style = MaterialTheme.typography.titleMedium, modifier = Modifier.padding(top = 4.dp))
}

@Composable
internal fun Pill(
    text: String,
    color: Color = MobileColors.Ocean,
    background: Color = MobileColors.Ice,
) {
    Text(
        text,
        color = color,
        style = MaterialTheme.typography.labelLarge,
        modifier = Modifier.clip(CircleShape).background(background).padding(horizontal = 11.dp, vertical = 5.dp),
    )
}

@Composable
internal fun IconTile(icon: ImageVector, background: Color, tint: Color = Color.White) {
    Box(Modifier.size(44.dp).clip(IconShape).background(background), contentAlignment = Alignment.Center) {
        Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(22.dp))
    }
}

@Composable
internal fun ServiceRow(
    title: String,
    subtitle: String,
    icon: ImageVector,
    color: Color = MobileColors.Neutral,
    iconTint: Color = MobileColors.Muted,
    trailing: String? = null,
    onClick: (() -> Unit)? = null,
) {
    val content: @Composable () -> Unit = {
        Row(
            Modifier.fillMaxWidth().heightIn(min = 64.dp).padding(vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconTile(icon, color, iconTint)
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(title, style = MaterialTheme.typography.bodyLarge)
                Text(subtitle, color = MobileColors.Muted, style = MaterialTheme.typography.bodyMedium)
            }
            if (trailing != null) {
                Text(trailing, style = MaterialTheme.typography.labelLarge)
            }
            if (onClick != null) {
                Icon(Icons.Filled.KeyboardArrowRight, contentDescription = null, tint = MobileColors.Muted)
            }
        }
    }
    if (onClick == null) {
        content()
    } else {
        Surface(onClick = onClick, shape = IconShape, color = Color.Transparent, content = content)
    }
}

@Composable
internal fun InfoCard(
    modifier: Modifier = Modifier,
    color: Color = MobileColors.Surface,
    onClick: (() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val body: @Composable () -> Unit = {
        Column(
            Modifier.fillMaxWidth().padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(7.dp),
            content = content,
        )
    }
    if (onClick == null) {
        Surface(modifier = modifier, shape = CardShape, color = color, content = body)
    } else {
        Surface(onClick = onClick, modifier = modifier, shape = CardShape, color = color, content = body)
    }
}
