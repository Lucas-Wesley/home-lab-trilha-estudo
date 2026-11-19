<script setup>
// Ícones de traço (16 px, traço 1,5), desenhados à mão no estilo da referência.
defineProps({
  nome: { type: String, required: true },
  tamanho: { type: Number, default: 16 },
})

const CAMINHOS = {
  marca: ['M3 20h5v-5h5v-5h5V5h3'],
  lupa: ['M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14Z', 'm20 20-4-4'],
  menu: ['M4 7h16', 'M4 12h16', 'M4 17h16'],
  fechar: ['M6 6l12 12', 'M18 6 6 18'],
  sol: ['M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z', 'M12 2v2', 'M12 20v2', 'm4.9 4.9 1.4 1.4', 'm17.7 17.7 1.4 1.4', 'M2 12h2', 'M20 12h2', 'm4.9 19.1 1.4-1.4', 'm17.7 6.3 1.4-1.4'],
  lua: ['M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5Z'],
  auto: ['M3 5h18v12H3z', 'M8 21h8', 'M12 17v4'],
  check: ['M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z', 'm8.5 12 2.5 2.5 4.5-5'],
  copiar: ['M9 9h10v10H9z', 'M5 15V5h10'],
  'seta-baixo': ['m7 10 5 5 5-5'],
  'seta-direita': ['m10 7 5 5-5 5'],
  mais: ['M12 5v14', 'M5 12h14'],
  ajustes: ['M4 6h9', 'M17 6h3', 'M15 4v4', 'M4 12h3', 'M11 12h9', 'M9 10v4', 'M4 18h11', 'M19 18h1', 'M17 16v4'],
  livro: ['M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2V5Z', 'M4 19a2 2 0 0 1 2-2h13'],
  video: ['M3 6h13v12H3z', 'm16 10 5-3v10l-5-3'],
  link: ['M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1', 'M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1'],
  comentario: ['M4 5h16v11H9l-5 4V5Z'],
  trilhas: ['M5 19V9', 'M12 19V5', 'M19 19v-7'],
  entrada: ['M12 3v11', 'm7 9 5 5 5-5', 'M4 17v3a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-3'],
  hoje: ['M4 6h16v14H4z', 'M4 10h16', 'M9 3v4', 'M15 3v4'],
  design: ['M12 21a9 9 0 1 1 0-18c5 0 9 3.6 9 8 0 2.5-2 4-4.5 4H15a2 2 0 0 0-1.5 3.3A1.6 1.6 0 0 1 12 21Z', 'M7.5 11.5h.01', 'M10.5 7.5h.01', 'M15.5 8.5h.01'],
}
</script>

<template>
  <svg
    :width="tamanho"
    :height="tamanho"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-for="(d, i) in CAMINHOS[nome] || []" :key="i" :d="d" />
  </svg>
</template>
