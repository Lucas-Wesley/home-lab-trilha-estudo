<script setup>
// Layout do design system (quarta versão): a navegação da página vai para a lateral do app (no
// celular, para a gaveta do item Menu); o cabeçalho (#topo) fica na largura toda; o conteúdo fica
// largo e o painel de apoio (#lateral) ao lado.
</script>

<template>
  <div class="pagina">
    <Teleport defer to="#nav-contexto"><slot name="nav" /></Teleport>
    <slot name="topo" />
    <div v-if="$slots.lateral" class="duas-colunas">
      <div><slot /></div>
      <aside class="lateral"><slot name="lateral" /></aside>
    </div>
    <slot v-else />
  </div>
</template>
