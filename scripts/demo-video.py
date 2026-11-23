"""Capture real demo API responses and render a narrated-by-captions MP4.

Requires Pillow and ffmpeg. Run against a freshly seeded isolated demo database.
"""
import argparse
import json
import subprocess
import time
import urllib.request
import uuid
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--url', default='http://127.0.0.1:18080')
    parser.add_argument('--output', default='artifacts/demo-video')
    args = parser.parse_args()
    out = Path(args.output)
    out.mkdir(parents=True, exist_ok=True)
    evidence = []

    def call(path, body=None):
        req = urllib.request.Request(args.url + path,
            data=json.dumps(body).encode() if body else None,
            headers={'Content-Type': 'application/json'})
        with urllib.request.urlopen(req, timeout=5) as response:
            result = json.load(response)
        evidence.append(dict(time=time.strftime('%Y-%m-%dT%H:%M:%S%z'),
                             path=path, request=body, response=result))
        return result

    results = {}
    for scenario in ['fraud', 'legal']:
        account = 'demo-' + scenario
        before = call('/v1/state/' + account)
        operation = str(uuid.uuid4())
        decision = call('/v1/evaluate', dict(accountId=account, operationId=operation,
            target='sink-a', device='new' if scenario == 'fraud' else 'known',
            attestation='demo-valid'))
        assert decision['result'] == ('hold' if scenario == 'fraud' else 'step_up'), decision
        pending = call('/v1/state/' + account)
        assert pending['changes'] == before['changes']
        allowed = None
        if scenario == 'legal':
            allowed = call('/v1/confirm', dict(accountId=account, operationId=operation,
                confirmationId=decision['confirmationId'], proof='demo-approved'))
            assert allowed['result'] == 'allow', allowed
            for _ in range(60):
                state = call('/v1/state/' + account)
                if state['changes'] == before['changes'] + 1:
                    break
                time.sleep(.1)
            assert state['changes'] == before['changes'] + 1 and state['target'] == 'sink-a'
            assert any(a['action'] == 'execution' and a['details']['result'] == 'executed'
                       for a in state['audit'])
        else:
            state = pending
            assert state['target'] == before['target']
        assert any(d['operation_id'] == operation and d['policy_version'] == 'demo-v1'
                   and d['evidence'] for d in state['decisions'])
        results[scenario] = dict(before=before, decision=decision, state=state)
    (out / 'api-evidence.json').write_text(json.dumps(evidence, ensure_ascii=False, indent=2), encoding='utf-8')
    fraud, legal = results['fraud'], results['legal']
    scenes = [
        ('СЕССИОННЫЙ АНТИФРОД', 'Защита переадресации',
         ['Два сценария на работающем сервере', 'Подозрительная операция и легитимный запрос', 'Визуализация реального API-прогона'],
         'Учебный стенд • только синтетические данные', 'PLATFORM DEMO', 8),
        ('01 / ПЛАТФОРМА', 'От сигнала до исполнения',
         ['Collector → Gateway → Risk → PostgreSQL', 'Сетевые регистрации формируют признаки риска', 'Решение и аудит сохраняются до доставки команды'],
         'Услугу исполняет локальный симулятор', 'demo-v1', 9),
        ('02 / ПОДОЗРИТЕЛЬНЫЙ ЗАПРОС', 'Новая SIM. Новое устройство.',
         ['Аккаунт: demo-fraud • назначение: sink-a', 'Калининград → Владивосток', 'POST /v1/evaluate → result: ' + fraud['decision']['result']],
         'Синтетический сценарий невозможного перемещения', 'RISK SIGNALS', 9),
        ('03 / ЗАЩИТА СРАБОТАЛА', 'Операция приостановлена',
         ['Решение сервера: HOLD', 'Состояние услуги: ' + str(fraud['state']['target']),
          'Изменений до / после: ' + str(fraud['before']['changes']) + ' / ' + str(fraud['state']['changes'])],
         'Переадресация не изменилась • охлаждение на 5 минут', 'HOLD', 9),
        ('04 / ЛЕГИТИМНЫЙ ЗАПРОС', 'Знакомое устройство',
         ['Аккаунт: demo-legal • назначение: sink-a', 'Калининград → Светлогорск • новая SIM',
          'POST /v1/evaluate → result: ' + legal['decision']['result']],
         'Перед изменением услуги требуется подтверждение', 'STEP UP', 9),
        ('05 / ПОДТВЕРЖДЕНИЕ', 'Разрешение получено',
         ['POST /v1/confirm → result: allow', 'Тестовый proof: demo-approved', 'Outbox доставляет команду после коммита'],
         'demo-approved — тестовая подстановка для этого стенда', 'ALLOW', 9),
        ('06 / ПРОВЕРКА НА СЕРВЕРЕ', 'Услуга действительно изменена',
         ['GET /v1/state/demo-legal → target: ' + str(legal['state']['target']),
          'Изменений до / после: ' + str(legal['before']['changes']) + ' / ' + str(legal['state']['changes']),
          'В аудите: execution → executed'],
         'Результат подтверждён ответом API и проверками сценария', 'EXECUTED', 9),
        ('РЕЗУЛЬТАТ ДЕМОНСТРАЦИИ', 'Риск проверен. Действие защищено.',
         ['demo-fraud → HOLD → услуга без изменений', 'demo-legal → STEP UP → ALLOW → sink-a',
          'Ответы сервера сохранены в api-evidence.json'],
         'Без операторской сети • без реальных абонентов • без звука', '2 / 2 PASSED', 8),
    ]
    regular = 'C:/Windows/Fonts/arial.ttf'
    bold = 'C:/Windows/Fonts/arialbd.ttf'
    font = lambda size, heavy=False: ImageFont.truetype(bold if heavy else regular, size)
    width, height, fps = 1920, 1080, 24
    movie = out / 'antifraud-demo.mp4'
    encoder = subprocess.Popen(['ffmpeg', '-y', '-loglevel', 'error', '-f', 'rawvideo',
        '-vcodec', 'rawvideo', '-pix_fmt', 'rgb24', '-s', f'{width}x{height}', '-r', str(fps),
        '-i', '-', '-an', '-c:v', 'libx264', '-preset', 'fast', '-crf', '20',
        '-pix_fmt', 'yuv420p', '-movflags', '+faststart', str(movie)], stdin=subprocess.PIPE)
    total = sum(s[-1] for s in scenes)
    elapsed = 0
    for index, (label, title, lines, footer, badge, duration) in enumerate(scenes):
        accent = '#ffb35b' if index in (2, 3) else '#58dfc5'
        base = Image.new('RGB', (width, height), '#0b1423')
        draw = ImageDraw.Draw(base)
        draw.rectangle((0, 0, 16, height), fill=accent)
        draw.text((110, 85), 'ANTIFRAUD / DEMO LAB', font=font(26, True), fill='#8ba0ba')
        draw.rounded_rectangle((1480, 72, 1810, 126), radius=24, fill='#1b3043')
        draw.text((1510, 85), 'СИНТЕТИЧЕСКИЕ ДАННЫЕ', font=font(18, True), fill=accent)
        draw.text((110, 230), label, font=font(27, True), fill=accent)
        title_size = 64
        while draw.textlength(title, font=font(title_size, True)) > 1700:
            title_size -= 2
        draw.text((105, 300), title, font=font(title_size, True), fill='#f4f7fb')
        draw.rounded_rectangle((110, 440, 1810, 785), radius=24, fill='#132236', outline='#29415a', width=2)
        for row, line in enumerate(lines):
            draw.ellipse((150, 490 + row * 95, 160, 500 + row * 95), fill=accent)
            draw.text((190, 472 + row * 95), line, font=font(34), fill='#dae4f0')
        draw.text((110, 830), footer, font=font(27), fill='#9cacc1')
        draw.text((110, 943), badge, font=font(28, True), fill=accent)
        draw.text((1710, 950), f'{index+1:02} / 08', font=font(24), fill='#9cacc1')
        base.save(out / f'scene-{index+1:02}.png')
        for frame in range(duration * fps):
            img = base.copy()
            d = ImageDraw.Draw(img)
            progress = (elapsed + frame / fps) / total
            d.rectangle((110, 1010, 1810, 1016), fill='#24394f')
            d.rectangle((110, 1010, 110 + int(1700 * progress), 1016), fill=accent)
            encoder.stdin.write(img.tobytes())
        elapsed += duration
        print(f'Scene {index+1}/8 rendered', flush=True)
    encoder.stdin.close()
    assert encoder.wait() == 0, 'ffmpeg failed'
    (out / 'README.txt').write_text('Демо платформы: 70 секунд, 1920×1080, 24 fps, без звука.\n'
        'Это визуализация реальных ответов API, а не запись Android-интерфейса.\n'
        'api-evidence.json содержит запросы и ответы из изолированной синтетической БД.\n'
        'Повторить: python scripts/demo-video.py --url http://127.0.0.1:18080\n', encoding='utf-8')
    print(movie.resolve())


if __name__ == '__main__':
    main()
