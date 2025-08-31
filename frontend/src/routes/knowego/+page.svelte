<script lang="ts">
  import { goto } from '$app/navigation';
  import { _, t } from 'svelte-i18n';
  import { preferencesStore, setTheme } from '$lib/stores/preferences.svelte.ts';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import { getAppLogo } from '$lib/config';
  import { 
    Youtube, Cpu, Database, Lock, Shield, Terminal, Settings, Layers, Image, Code, Brain, Eye, Bot, BookCopy, Target, Zap, ArrowRight, Play, Pause, RotateCcw, CheckCircle, Workflow, Sparkles, Rocket, Users, TrendingUp, Award, Globe, Mail, Phone, MapPin, Github, Twitter, Linkedin, Sun, Moon, Activity, Calculator, BookOpen, Book, Upload, FileText, Shuffle 
  } from '@lucide/svelte';
  import LoadingButton from '$lib/components/LoadingButton.svelte';
  import { withLoading } from '$lib/stores/loading.svelte.ts';
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api';
  import type { PublicStats } from '$lib/types';
  function toggleTheme() {
    setTheme(preferencesStore.theme === 'light' ? 'dark' : 'light');
  }
  let mounted = $state(false);
  let currentStep = $state(0);
  let isPlaying = $state(false);
  let demoComplete = $state(false);
  let demoText = $state("");
  let typingIndex = $state(0);
  let isTyping = $state(false);
  let heroVisible = $state(false);
  let currentScenario = $state(0);
  let showOutput = $state(false);
  let outputContent = $state("");
  let publicStats = $state({
    total_tokens: 15420000,
    total_sessions: 89250,
    total_requests: 1250000,
    avg_response_time_ms: 1250,
    days_active: 127,
    total_files_uploaded: 125000,
    total_thinking_iterations: 2500000
  });
  const nf = new Intl.NumberFormat('ru-RU');
  const formatNumber = (n: number | undefined | null) => nf.format(Number(n) || 0);
  const demoScenarios = [
    {
      title: $t('knowego.demo_scenario_code'),
      steps: [
        { text: $t('knowego.demo_step_upload'), icon: 'Upload', output: "def calculate_fibonacci(n):\n    if n <= 1:\n        return n\n    return calculate_fibonacci(n-1) + calculate_fibonacci(n-2)\n\nresult = calculate_fibonacci(50)  # This will be slow!" },
        { text: $t('knowego.demo_step_analyze'), icon: 'Brain', output: null },
        { text: $t('knowego.demo_step_test'), icon: 'Terminal', output: "Testing optimized version...\n✓ Execution time: 0.001s (was: 30s+)\n✓ Memory usage: 8KB (was: 2GB+)" },
        { text: $t('knowego.demo_step_solution'), icon: 'Zap', output: "from functools import lru_cache\n\n@lru_cache(maxsize=None)\ndef calculate_fibonacci(n):\n    if n <= 1:\n        return n\n    return calculate_fibonacci(n-1) + calculate_fibonacci(n-2)" },
        { text: $t('knowego.demo_step_explain'), icon: 'Sparkles', output: null }
      ]
    },
    {
      title: $t('knowego.demo_scenario_research'),
      steps: [
        { text: $t('knowego.demo_step_research_ask'), icon: 'Brain', output: null },
        { text: $t('knowego.demo_step_research_search'), icon: 'Globe', output: "Search: 'quantum computing 2024 breakthrough IBM Google'\nFound 847 relevant sources\nAnalyzing source credibility..." },
        { text: $t('knowego.demo_step_research_video'), icon: 'Youtube', output: "Processing video: 'IBM Quantum Network 2024'\nExtracting key insights\n45:30 → 3 min summary" },
        { text: $t('knowego.demo_step_research_calc'), icon: 'Calculator', output: "Quantum supremacy calculation:\n• Classical: 2^n operations\n• Quantum: √n operations\n• For n=300: 10^90 vs 17 operations" },
        { text: $t('knowego.demo_step_research_report'), icon: 'CheckCircle', output: null }
      ]
    },
    {
      title: $t('knowego.demo_scenario_creative'),
      steps: [
        { text: $t('knowego.demo_step_creative_request'), icon: 'Code', output: null },
        { text: $t('knowego.demo_step_creative_plan'), icon: 'Brain', output: "Requirements analysis:\n• Frontend: React + TypeScript\n• Backend: Node.js + Express\n• Database: PostgreSQL\n• Deploy: Docker + Kubernetes" },
        { text: $t('knowego.demo_step_creative_generate'), icon: 'Terminal', output: "interface User {\n  id: string;\n  name: string;\n  email: string;\n}\n\nconst UserCard: React.FC<{user: User}> = ({user}) => {\n  return (\n    <div className=\"user-card\">\n      <h3>{user.name}</h3>\n      <p>{user.email}</p>\n    </div>\n  );\n};" },
        { text: $t('knowego.demo_step_creative_review'), icon: 'Users', output: "User: \"Interface is intuitive\"\nManager: \"Need analytics dashboard\"\nDeveloper: \"Code is clean and scalable\"" },
        { text: $t('knowego.demo_step_creative_deploy'), icon: 'Rocket', output: null }
      ]
    }
  ];
  const demoSteps = $derived(demoScenarios[currentScenario]?.steps || []);
  async function fetchPublicStats() {
    try {
      const data = await api.get<PublicStats>('/public-stats');
      publicStats = data;
      console.log('Public stats loaded:', data);
    } catch (error) {
      console.log('Using fallback stats:', error);
    }
  }
  onMount(() => {
    mounted = true;
    fetchPublicStats();
    setTimeout(() => {
      heroVisible = true;
      startDemo();
    }, 500);
  });
  onDestroy(() => {
    mounted = false;
    isPlaying = false;
  });
  function startDemo() {
    if (!mounted || isPlaying) return;
    isPlaying = true;
    currentStep = 0;
    demoText = "";
    demoComplete = false;
    const stepDelays = [1500, 2000, 1800, 1200, 1500];
    let stepIndex = 0;
    function nextStep() {
      if (!isPlaying || !mounted) return;
      if (stepIndex >= demoSteps.length) {
        demoComplete = true;
        setTimeout(() => {
          if (isPlaying) restartDemo();
        }, 5000);
        return;
      }
      currentStep = stepIndex;
      typeText(demoSteps[stepIndex]?.text || '', () => {
        const currentStepData = demoSteps[stepIndex];
        const delay = currentStepData?.output ? 3000 : 1500; 
        setTimeout(() => {
          stepIndex++;
          nextStep();
        }, delay);
      });
    }
    nextStep();
  }
  function typeText(text: string, onComplete: () => void) {
    isTyping = true;
    typingIndex = 0;
    demoText = "";
    showOutput = false;
    const interval = setInterval(() => {
      if (!isPlaying || !mounted) {
        clearInterval(interval);
        return;
      }
      if (typingIndex < text.length) {
        demoText += text[typingIndex];
        typingIndex++;
      } else {
        clearInterval(interval);
        isTyping = false;
        const currentStepData = demoSteps[currentStep];
        if (currentStepData?.output) {
          setTimeout(() => {
            outputContent = currentStepData.output;
            showOutput = true;
          }, 800);
        }
        onComplete();
      }
    }, 30);
  }
  function switchScenario(scenarioIndex: number) {
    if (scenarioIndex === currentScenario) return;
    isPlaying = false;
    currentScenario = scenarioIndex;
    currentStep = 0;
    demoText = "";
    showOutput = false;
    outputContent = "";
    setTimeout(() => {
      startDemo();
    }, 300);
  }
  function restartDemo() {
    isPlaying = false;
    setTimeout(() => {
      if (mounted) startDemo();
    }, 100);
  }
  function toggleDemo() {
    if (isPlaying) {
      isPlaying = false;
    } else {
      startDemo();
    }
  }
</script>
<svelte:head>
  <title>{$_('knowego.title')}</title>
  <meta name="description" content={$_('knowego.subtitle')} />
</svelte:head>
<div class="relative w-full min-h-screen overflow-hidden bg-primary">
  <div class="bg-orb pointer-events-none">
    <div class="bg-circle" style="--dur: 35s; top: 15%; left: 10%;"></div>
    <div class="bg-circle secondary" style="--dur: 42s; top: 60%; right: 15%;"></div>
    <div class="bg-circle" style="--dur: 38s; top: 80%; left: 70%; transform: scale(0.6);"></div>
  </div>
  <nav class="absolute top-0 left-0 right-0 z-50 p-4 sm:p-6">
    <div class="max-w-7xl mx-auto flex items-center justify-between">
      <div class="flex items-center gap-2 sm:gap-4">
        <div class="relative group">
          <div class="absolute -inset-1 bg-gradient-to-r from-accent to-accent-hover rounded-lg blur opacity-25 group-hover:opacity-40 transition duration-300"></div>
          <div class="relative bg-secondary/90 backdrop-blur-sm rounded-lg p-1.5 sm:p-2 border border-tertiary/50">
            <img src={getAppLogo(preferencesStore.theme)} alt={$_('sidebar.title')} class="w-5 h-5 sm:w-6 sm:h-6" />
          </div>
        </div>
        <span class="text-lg sm:text-xl font-bold text-text-primary tracking-tight hidden xs:block">{$_('sidebar.title')}</span>
      </div>
      <div class="flex items-center gap-2 sm:gap-4">
        <div class="bg-secondary/60 backdrop-blur-sm rounded-full p-0.5 sm:p-1 border border-tertiary/30">
          <LanguageSwitcher />
        </div>
        <button 
          onclick={toggleTheme} 
          class="group relative p-2 sm:p-3 rounded-full bg-secondary/60 backdrop-blur-sm border border-tertiary/30 hover:border-accent/50 transition-all duration-500 hover:scale-110 hover:rotate-12 touch-manipulation"
          aria-label={$t('knowego.nav_toggle_theme')}
        >
          <div class="absolute inset-0 rounded-full bg-gradient-to-r from-accent/20 to-accent-hover/20 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          {#if preferencesStore.theme === 'light'}
            <Moon class="w-4 h-4 sm:w-5 sm:h-5 text-text-primary relative z-10 transition-transform duration-300 group-hover:scale-110" />
          {:else}
            <Sun class="w-4 h-4 sm:w-5 sm:h-5 text-text-primary relative z-10 transition-transform duration-300 group-hover:scale-110" />
          {/if}
        </button>
      </div>
    </div>
  </nav>
  <section class="relative px-4 sm:px-6 pt-24 pb-16 sm:pt-32 sm:pb-20 md:pt-40 md:pb-28">
    <div class="max-w-7xl mx-auto text-center">
      <div class="transition-all duration-1000 delay-500 {heroVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'}">
        <h1 class="text-4xl sm:text-5xl md:text-7xl lg:text-8xl font-black mb-6 sm:mb-8 text-text-primary leading-[0.9] tracking-tight px-2">
          <span class="bg-gradient-to-r from-accent via-text-primary to-accent-hover bg-clip-text text-transparent animate-gradient-x bg-[length:200%_200%]">
            {$t('knowego.hero_meet_ego')}
          </span>
        </h1>
        <p class="text-lg sm:text-xl md:text-2xl text-text-secondary mb-8 sm:mb-10 max-w-4xl mx-auto leading-relaxed font-medium px-4">{$t('knowego.hero_description')}</p>
      </div>
      <div class="mt-12 sm:mt-16 transition-all duration-1000 delay-700 {heroVisible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'} px-4">
        <div class="group relative inline-block w-full sm:w-auto">
          <div class="absolute -inset-1 bg-gradient-to-r from-accent via-accent-hover to-accent rounded-2xl blur-lg opacity-50 group-hover:opacity-75 transition duration-500"></div>
          <button
            onclick={async () => {
              await withLoading(
                async () => {
                  await new Promise(resolve => setTimeout(resolve, 1000));
                  goto('/chat/new');
                },
                $_('chat.creating_session'),
                'sparkles'
              );
            }}
            class="relative bg-gradient-to-r from-accent to-accent-hover hover:from-accent-hover hover:to-accent text-white font-bold px-8 sm:px-12 py-4 sm:py-5 rounded-2xl shadow-2xl shadow-accent/30 hover:shadow-accent/50 transition-all duration-500 hover:scale-105 active:scale-95 text-base sm:text-lg flex items-center justify-center gap-3 w-full sm:w-auto touch-manipulation"
          >
            <Rocket class="w-5 h-5 sm:w-6 sm:h-6 group-hover:rotate-12 transition-transform duration-300" />
            <span class="whitespace-nowrap">{$t('knowego.hero_join_beta')}</span>
            <ArrowRight class="w-5 h-5 sm:w-6 sm:h-6 group-hover:translate-x-2 transition-transform duration-300" />
          </button>
        </div>
      </div>
    </div>
    <div id="features" class="max-w-7xl mx-auto mt-20 sm:mt-32 px-4 sm:px-6">
        <div class="text-center mb-12 sm:mb-16">
          <h3 class="text-2xl sm:text-3xl md:text-4xl font-bold text-text-primary mb-3 sm:mb-4 px-4">{$t('knowego.features_section_title')}</h3>
          <p class="text-base sm:text-lg text-text-secondary px-4">{$t('knowego.features_section_subtitle')}</p>
        </div>
        <div class="max-w-4xl mx-auto grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6 lg:gap-8 justify-items-stretch">
            <div class="group relative text-center p-6 sm:p-8 rounded-2xl bg-secondary/60 backdrop-blur-sm border border-tertiary/30 transition-all duration-500 hover:border-accent/50 hover:bg-secondary/80 hover:shadow-2xl hover:shadow-accent/20 hover:-translate-y-2 animate-fade-in-up touch-manipulation" style="animation-delay: 0.1s;">
                <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-accent/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
                <div class="relative z-10">
                    <div class="w-12 h-12 sm:w-16 sm:h-16 mx-auto mb-4 sm:mb-6 rounded-2xl bg-gradient-to-br from-accent/20 to-accent/10 border border-accent/30 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                        <Users class="w-6 h-6 sm:w-8 sm:h-8 text-accent"/>
                    </div>
                    <div class="text-2xl sm:text-3xl font-black text-text-primary mb-1 sm:mb-2">{formatNumber(publicStats.total_sessions)}</div>
                    <p class="text-xs sm:text-sm font-semibold text-text-secondary">{$t('knowego.total_sessions')}</p>
                </div>
            </div>
            <div class="group relative text-center p-6 sm:p-8 rounded-2xl bg-secondary/60 backdrop-blur-sm border border-tertiary/30 transition-all duration-500 hover:border-accent/50 hover:bg-secondary/80 hover:shadow-2xl hover:shadow-accent/20 hover:-translate-y-2 animate-fade-in-up touch-manipulation" style="animation-delay: 0.2s;">
                <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-accent/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
                <div class="relative z-10">
                    <div class="w-12 h-12 sm:w-16 sm:h-16 mx-auto mb-4 sm:mb-6 rounded-2xl bg-gradient-to-br from-accent/20 to-accent/10 border border-accent/30 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                        <TrendingUp class="w-6 h-6 sm:w-8 sm:h-8 text-accent"/>
                    </div>
                    <div class="text-2xl sm:text-3xl font-black text-text-primary mb-1 sm:mb-2">{formatNumber(publicStats.total_tokens)}</div>
                    <p class="text-xs sm:text-sm font-semibold text-text-secondary">{$t('knowego.total_tokens')}</p>
                </div>
            </div>
            <div class="group relative text-center p-6 sm:p-8 rounded-2xl bg-secondary/60 backdrop-blur-sm border border-tertiary/30 transition-all duration-500 hover:border-accent/50 hover:bg-secondary/80 hover:shadow-2xl hover:shadow-accent/20 hover:-translate-y-2 animate-fade-in-up touch-manipulation" style="animation-delay: 0.3s;">
                <div class="absolute inset-0 rounded-2xl bg-gradient-to-br from-accent/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
                <div class="relative z-10">
                    <div class="w-12 h-12 sm:w-16 sm:h-16 mx-auto mb-4 sm:mb-6 rounded-2xl bg-gradient-to-br from-accent/20 to-accent/10 border border-accent/30 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                        <Activity class="w-6 h-6 sm:w-8 sm:h-8 text-accent"/>
                    </div>
                    <div class="text-2xl sm:text-3xl font-black text-text-primary mb-1 sm:mb-2">{formatNumber(publicStats.total_requests)}</div>
                    <p class="text-xs sm:text-sm font-semibold text-text-secondary">{$t('knowego.total_requests')}</p>
                </div>
            </div>
    </div>
  </section>
  <section id="demo" class="px-4 sm:px-6 py-16 sm:py-20 md:py-32">
    <div class="max-w-5xl mx-auto">
      <div class="text-center mb-12 sm:mb-16">
        <h2 class="text-3xl sm:text-4xl md:text-5xl font-black text-text-primary mb-4 sm:mb-6 px-4">
          <span class="bg-gradient-to-r from-text-primary via-accent to-text-primary bg-clip-text text-transparent animate-gradient-x">
            {$t('knowego.demo_what_makes_different')}
          </span>
        </h2>
        <p class="text-lg sm:text-xl text-text-secondary max-w-3xl mx-auto leading-relaxed px-4">{$t('knowego.demo_different_subtitle')}</p>
      </div>
      <div class="flex flex-wrap items-center justify-center gap-3 mb-8">
        {#each demoScenarios as scenario, index}
          <button
            onclick={() => switchScenario(index)}
            class="group relative px-4 py-2.5 text-sm font-medium rounded-xl border transition-all duration-300 touch-manipulation {currentScenario === index ? 'bg-gradient-to-r from-accent to-accent-hover text-white border-accent shadow-lg shadow-accent/30' : 'bg-secondary/60 text-text-secondary border-tertiary/30 hover:border-accent/50 hover:bg-secondary/80'}"
          >
            <span class="relative z-10">{scenario.title}</span>
            {#if currentScenario === index}
              <div class="absolute inset-0 rounded-xl bg-gradient-to-r from-accent/20 to-accent-hover/20 blur opacity-50"></div>
            {/if}
          </button>
        {/each}
      </div>
      <div class="group relative rounded-2xl sm:rounded-3xl border border-tertiary/30 bg-secondary/60 backdrop-blur-sm shadow-2xl shadow-black/20 hover:shadow-accent/10 transition-all duration-500 max-w-4xl mx-auto overflow-hidden">
        <div class="absolute inset-0 rounded-2xl sm:rounded-3xl bg-gradient-to-br from-accent/5 via-transparent to-accent/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
        <div class="relative z-10 p-6 sm:p-8 md:p-10">
          <div class="flex items-center gap-3 mb-6">
            <div class="flex gap-2">
              <div class="w-3 h-3 rounded-full bg-red-500 shadow-lg shadow-red-500/50"></div>
              <div class="w-3 h-3 rounded-full bg-yellow-500 shadow-lg shadow-yellow-500/50"></div>
              <div class="w-3 h-3 rounded-full bg-green-500 shadow-lg shadow-green-500/50"></div>
            </div>
            <span class="text-sm text-text-secondary font-mono font-semibold tracking-wide">{$t('knowego.demo_reasoning_engine')}</span>
            <div class="ml-auto flex gap-2">
              <button onclick={restartDemo} class="group/btn p-1.5 rounded-lg bg-tertiary/30 hover:bg-tertiary/60 border border-tertiary/20 hover:border-accent/50 transition-all duration-300 touch-manipulation" title={$t('knowego.demo_restart')}>
                <RotateCcw class="w-3.5 h-3.5 text-text-secondary group-hover/btn:text-accent transition-colors duration-300" />
              </button>
              <button onclick={toggleDemo} class="group/btn p-1.5 rounded-lg bg-tertiary/30 hover:bg-tertiary/60 border border-tertiary/20 hover:border-accent/50 transition-all duration-300 touch-manipulation" title={isPlaying ? $t('knowego.demo_pause') : $t('knowego.demo_play')}>
                {#if isPlaying}
                  <Pause class="w-3.5 h-3.5 text-text-secondary group-hover/btn:text-accent transition-colors duration-300" />
                {:else}
                  <Play class="w-3.5 h-3.5 text-text-secondary group-hover/btn:text-accent transition-colors duration-300" />
                {/if}
              </button>
            </div>
          </div>
          <div class="bg-primary/80 backdrop-blur-sm rounded-xl p-6 sm:p-8 border border-tertiary/40 shadow-inner min-h-[300px]">
            <div class="flex items-start gap-4 mb-6">
              {#if currentStep < demoSteps.length && demoSteps[currentStep]}
                <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-accent/20 to-accent/10 border border-accent/30 flex items-center justify-center flex-shrink-0 {isPlaying ? 'animate-pulse shadow-lg shadow-accent/30' : 'shadow-md shadow-accent/20'}">
                  {#if demoSteps[currentStep].icon === 'Brain'} <Brain class="w-5 h-5 text-accent" />
                  {:else if demoSteps[currentStep].icon === 'Zap'} <Zap class="w-5 h-5 text-yellow-400" />
                  {:else if demoSteps[currentStep].icon === 'CheckCircle'} <CheckCircle class="w-5 h-5 text-green-400" />
                  {:else if demoSteps[currentStep].icon === 'Sparkles'} <Sparkles class="w-5 h-5 text-accent" />
                  {:else if demoSteps[currentStep].icon === 'Terminal'} <Terminal class="w-5 h-5 text-green-400" />
                  {:else if demoSteps[currentStep].icon === 'Globe'} <Globe class="w-5 h-5 text-blue-400" />
                  {:else if demoSteps[currentStep].icon === 'Youtube'} <Youtube class="w-5 h-5 text-red-400" />
                  {:else if demoSteps[currentStep].icon === 'Calculator'} <Calculator class="w-5 h-5 text-orange-400" />
                  {:else if demoSteps[currentStep].icon === 'Code'} <Code class="w-5 h-5 text-cyan-400" />
                  {:else if demoSteps[currentStep].icon === 'Users'} <Users class="w-5 h-5 text-purple-400" />
                  {:else if demoSteps[currentStep].icon === 'Upload'} <Upload class="w-5 h-5 text-indigo-400" />
                  {:else if demoSteps[currentStep].icon === 'Rocket'} <Rocket class="w-5 h-5 text-pink-400" />
                  {/if}
                </div>
                <div class="flex-1 min-w-0">
                  <div class="text-text-primary text-base font-medium leading-relaxed">
                    {demoText}
                    {#if isTyping}
                      <span class="inline-block w-2 h-5 bg-accent ml-1 animate-pulse"></span>
                    {/if}
                  </div>
                </div>
              {/if}
            </div>
            {#if showOutput && outputContent}
              <div class="mt-6 p-4 bg-tertiary/20 rounded-lg border border-tertiary/30 transition-all duration-500 animate-fade-in-up">
                <div class="flex items-center gap-2 mb-3">
                  <div class="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                  <span class="text-sm text-text-secondary font-semibold">Output:</span>
                </div>
                <pre class="text-sm text-text-primary font-mono whitespace-pre-wrap leading-relaxed overflow-x-auto">{outputContent}</pre>
              </div>
            {/if}
            <div class="flex items-center justify-center mt-8">
              <div class="flex gap-2">
                {#each demoSteps as step, index}
                  <div class="relative">
                    <div class="w-3 h-3 rounded-full transition-all duration-300 {index <= currentStep ? 'bg-accent shadow-lg shadow-accent/50' : 'bg-tertiary/50'} {index === currentStep && isPlaying ? 'animate-pulse' : ''}"></div>
                    {#if index <= currentStep && isPlaying && index === currentStep}
                      <div class="absolute -top-1 -left-1 w-5 h-5 rounded-full bg-accent/20 animate-ping"></div>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
  <section class="px-4 sm:px-6 py-12 sm:py-16 md:py-20 bg-gradient-to-br from-accent/5 via-secondary/10 to-accent/5">
    <div class="max-w-6xl mx-auto text-center">
      <div class="relative group">
        <div class="absolute -inset-1 bg-gradient-to-r from-accent to-accent-hover rounded-3xl blur-lg opacity-20 group-hover:opacity-30 transition duration-500"></div>
        <div class="relative bg-secondary/80 backdrop-blur-sm rounded-3xl p-8 sm:p-12 border border-tertiary/30 shadow-2xl">
          <div class="flex items-center justify-center gap-3 mb-6">
            <div class="w-12 h-12 rounded-2xl bg-gradient-to-br from-green-500/20 to-emerald-500/10 flex items-center justify-center border border-green-500/30">
              <TrendingUp class="w-6 h-6 text-green-400" />
            </div>
            <h2 class="text-3xl sm:text-4xl md:text-5xl font-black text-text-primary">
              <span class="bg-gradient-to-r from-green-400 via-accent to-emerald-400 bg-clip-text text-transparent animate-gradient-x bg-[length:200%_200%]">
                {$t('knowego.pricing_advantage_title')}
              </span>
            </h2>
          </div>
          <p class="text-lg sm:text-xl text-text-secondary mb-8 max-w-3xl mx-auto leading-relaxed">{$t('knowego.pricing_advantage_subtitle')}</p>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-6 sm:gap-8">
            <div class="group relative p-6 rounded-2xl bg-primary/60 border border-tertiary/30 hover:border-green-400/50 transition-all duration-300 hover:shadow-lg hover:shadow-green-400/20">
              <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-green-500/20 to-green-500/10 flex items-center justify-center mx-auto mb-4 border border-green-500/30">
                <Target class="w-5 h-5 text-green-400" />
              </div>
              <h3 class="text-lg font-bold text-text-primary mb-2">{$t('knowego.pricing_advantage_cost')}</h3>
              <p class="text-sm text-text-secondary">vs GPT-5 & Claude 4.1 Opus</p>
            </div>
            <div class="group relative p-6 rounded-2xl bg-primary/60 border border-tertiary/30 hover:border-accent/50 transition-all duration-300 hover:shadow-lg hover:shadow-accent/20">
              <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-accent/20 to-accent/10 flex items-center justify-center mx-auto mb-4 border border-accent/30">
                <Zap class="w-5 h-5 text-accent" />
              </div>
              <h3 class="text-lg font-bold text-text-primary mb-2">{$t('knowego.pricing_advantage_tokens')}</h3>
              <p class="text-sm text-text-secondary">1M+ context window</p>
            </div>
            <div class="group relative p-6 rounded-2xl bg-primary/60 border border-tertiary/30 hover:border-purple-400/50 transition-all duration-300 hover:shadow-lg hover:shadow-purple-400/20">
              <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-purple-500/20 to-purple-500/10 flex items-center justify-center mx-auto mb-4 border border-purple-500/30">
                <Award class="w-5 h-5 text-purple-400" />
              </div>
              <h3 class="text-lg font-bold text-text-primary mb-2">{$t('knowego.pricing_advantage_quality')}</h3>
              <p class="text-sm text-text-secondary">Latest Gemini 2.5 Flash</p>
            </div>
          </div>
          <p class="text-sm sm:text-base text-text-secondary mt-8 max-w-2xl mx-auto leading-relaxed">{$t('knowego.pricing_advantage_desc')}</p>
        </div>
      </div>
    </div>
  </section>
  <section class="px-4 sm:px-6 py-16 sm:py-20 md:py-32 bg-secondary/10">
    <div class="max-w-7xl mx-auto">
      <div class="text-center mb-12 sm:mb-16">
        <h2 class="text-3xl sm:text-4xl md:text-5xl font-black mb-4 sm:mb-6 px-4">
          <span class="bg-gradient-to-r from-text-primary via-accent to-text-primary bg-clip-text text-transparent animate-gradient-x bg-[length:200%_200%]">
            {$t('knowego.features_capabilities_title')}
          </span>
        </h2>
        <p class="text-lg sm:text-xl text-text-secondary max-w-3xl mx-auto leading-relaxed px-4">
          {$t('knowego.features_capabilities_subtitle')}
        </p>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-accent/5 via-transparent to-accent/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-blue-500/20 to-blue-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-blue-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-blue-500/20">
              <Globe class="w-6 h-6 sm:w-8 sm:h-8 text-blue-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_egosearch_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_egosearch_desc')}</p>
            <div class="inline-flex items-center gap-2 text-blue-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Zap class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_egosearch_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-red-500/5 via-transparent to-red-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-red-500/20 to-red-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-red-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-red-500/20">
              <Youtube class="w-6 h-6 sm:w-8 sm:h-8 text-red-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_egotube_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_egotube_desc')}</p>
            <div class="inline-flex items-center gap-2 text-red-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Eye class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_egotube_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-green-500/5 via-transparent to-green-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-green-500/20 to-green-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-green-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-green-500/20">
              <Terminal class="w-6 h-6 sm:w-8 sm:h-8 text-green-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_egocodeexec_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_egocodeexec_desc')}</p>
            <div class="inline-flex items-center gap-2 text-green-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Code class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_egocodeexec_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-purple-500/5 via-transparent to-purple-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-purple-500/20 to-purple-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-purple-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-purple-500/20">
              <Database class="w-6 h-6 sm:w-8 sm:h-8 text-purple-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_egomemory_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_egomemory_desc')}</p>
            <div class="inline-flex items-center gap-2 text-purple-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Brain class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_egomemory_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-orange-500/5 via-transparent to-orange-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-orange-500/20 to-orange-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-orange-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-orange-500/20">
              <Calculator class="w-6 h-6 sm:w-8 sm:h-8 text-orange-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_egocalc_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_egocalc_desc')}</p>
            <div class="inline-flex items-center gap-2 text-orange-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Calculator class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_egocalc_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-indigo-500/5 via-transparent to-indigo-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-indigo-500/20 to-indigo-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-indigo-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-indigo-500/20">
              <BookOpen class="w-6 h-6 sm:w-8 sm:h-8 text-indigo-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_egowiki_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_egowiki_desc')}</p>
            <div class="inline-flex items-center gap-2 text-indigo-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Book class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_egowiki_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-cyan-500/5 via-transparent to-cyan-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-cyan-500/20 to-cyan-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-cyan-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-cyan-500/20">
              <Users class="w-6 h-6 sm:w-8 sm:h-8 text-cyan-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_alterego_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_alterego_desc')}</p>
            <div class="inline-flex items-center gap-2 text-cyan-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Shuffle class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_alterego_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-emerald-500/5 via-transparent to-emerald-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-emerald-500/20 to-emerald-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-emerald-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-emerald-500/20">
              <Eye class="w-6 h-6 sm:w-8 sm:h-8 text-emerald-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_visible_thinking_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_visible_thinking_desc')}</p>
            <div class="inline-flex items-center gap-2 text-emerald-400 font-semibold w-full justify-center text-sm sm:text-base">
              <Brain class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_visible_thinking_badge')}
            </div>
          </div>
        </div>
        <div class="group relative rounded-2xl border border-tertiary/30 bg-secondary/60 p-6 sm:p-8 hover:shadow-xl hover:border-accent/50 transition-all duration-500 hover:-translate-y-2 touch-manipulation overflow-hidden">
          <div class="absolute inset-0 bg-gradient-to-br from-pink-500/5 via-transparent to-pink-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="relative z-10">
            <div class="w-12 h-12 sm:w-16 sm:h-16 rounded-2xl bg-gradient-to-br from-pink-500/20 to-pink-600/10 flex items-center justify-center mx-auto mb-4 sm:mb-6 border border-pink-500/30 group-hover:scale-110 transition-all duration-300 shadow-lg shadow-pink-500/20">
              <Upload class="w-6 h-6 sm:w-8 sm:h-8 text-pink-400" />
            </div>
            <h3 class="text-xl sm:text-2xl font-bold text-text-primary mb-3 sm:mb-4 text-center">{$t('knowego.tool_multimodal_title')}</h3>
            <p class="text-sm sm:text-base text-text-secondary leading-relaxed mb-3 sm:mb-4 text-center">{$t('knowego.tool_multimodal_desc')}</p>
            <div class="inline-flex items-center gap-2 text-pink-400 font-semibold w-full justify-center text-sm sm:text-base">
              <FileText class="w-3 h-3 sm:w-4 sm:h-4" />
              {$t('knowego.tool_multimodal_badge')}
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
  <footer class="px-4 sm:px-6 py-12 sm:py-16 border-t border-tertiary/30 bg-secondary/20">
    <div class="max-w-7xl mx-auto">
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-6 sm:gap-8 mb-8 sm:mb-12">
        <div class="col-span-1 sm:col-span-2">
          <div class="flex items-center gap-3 sm:gap-4 mb-4 sm:mb-6">
            <div class="relative group">
              <div class="absolute -inset-1 bg-gradient-to-r from-accent to-accent-hover rounded-lg blur opacity-25 group-hover:opacity-40 transition duration-300"></div>
              <div class="relative bg-secondary/90 backdrop-blur-sm rounded-lg p-1.5 sm:p-2 border border-tertiary/50">
                <img src={getAppLogo(preferencesStore.theme)} alt={$_('sidebar.title')} class="w-5 h-5 sm:w-6 sm:h-6" />
              </div>
            </div>
            <span class="text-lg sm:text-2xl font-bold text-text-primary tracking-tight">{$_('sidebar.title')}</span>
          </div>
          <p class="text-sm sm:text-base text-text-secondary mb-4 sm:mb-6 max-w-md">
            {$t('knowego.footer_experiment_description')}
          </p>
        </div>
        <div>
          <h4 class="font-bold text-text-primary mb-3 sm:mb-4 text-sm sm:text-base">{$t('knowego.footer_product')}</h4>
          <ul class="space-y-2 sm:space-y-3 text-text-secondary text-sm sm:text-base">
            <li><a href="#features" class="hover:text-accent transition-colors duration-300 touch-manipulation">{$t('knowego.footer_features')}</a></li>
            <li><a href="#demo" class="hover:text-accent transition-colors duration-300 touch-manipulation">{$t('knowego.footer_live_demo')}</a></li>
            <li><a href="/chat/new" class="hover:text-accent transition-colors duration-300 touch-manipulation">{$t('knowego.footer_try_ego')}</a></li>
          </ul>
        </div>
        <div>
          <h4 class="font-bold text-text-primary mb-3 sm:mb-4 text-sm sm:text-base">{$t('knowego.footer_legal')}</h4>
          <ul class="space-y-2 sm:space-y-3 text-text-secondary text-sm sm:text-base">
            <li><a href="/privacy" class="hover:text-accent transition-colors duration-300 touch-manipulation">{$t('knowego.footer_privacy_policy')}</a></li>
            <li><a href="/terms" class="hover:text-accent transition-colors duration-300 touch-manipulation">{$t('knowego.footer_terms_of_service')}</a></li>
          </ul>
        </div>
      </div>
      <div class="border-t border-tertiary/30 pt-6 sm:pt-8">
        <div class="flex flex-col md:flex-row items-center justify-between gap-3 sm:gap-4 mb-6 sm:mb-8">
          <p class="text-text-secondary text-xs sm:text-sm text-center md:text-left">
            {$t('knowego.footer_copyright')}
          </p>
        </div>
        <div class="space-y-2 sm:space-y-3 text-xs text-text-secondary/80 border-t border-tertiary/20 pt-4 sm:pt-6">
          <p>{$t('knowego.footer_disclaimer')}</p>
        </div>
      </div>
    </div>
  </footer>
</div>
<style>
  @keyframes gradient-x {
    0%, 100% {
      background-size: 200% 200%;
      background-position: left center;
    }
    50% {
      background-size: 200% 200%;
      background-position: right center;
    }
  }
  .animate-gradient-x {
    animation: gradient-x 4s ease infinite;
  }
  @keyframes spin-slow {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
  .animate-spin-slow {
    animation: spin-slow 10s linear infinite;
  }
  @keyframes fade-in-up {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  .animate-fade-in-up {
    animation: fade-in-up 0.6s ease-out forwards;
    opacity: 0; 
  }
  :global(html) {
    scroll-behavior: smooth;
  }
  button:focus-visible,
  a:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
  @media (max-width: 480px) {
    .xs\:block {
      display: block;
    }
  }
  @media (hover: none) and (pointer: coarse) {
    .touch-manipulation {
      touch-action: manipulation;
      -webkit-tap-highlight-color: rgba(0, 0, 0, 0);
    }
  }
  @keyframes float {
    0%, 100% { transform: translateY(0px); }
    50% { transform: translateY(-10px); }
  }
  @keyframes pulse-glow {
    0%, 100% { box-shadow: 0 0 20px rgba(var(--accent-rgb), 0.3); }
    50% { box-shadow: 0 0 30px rgba(var(--accent-rgb), 0.6); }
  }
  .animate-float {
    animation: float 6s ease-in-out infinite;
  }
  .animate-pulse-glow {
    animation: pulse-glow 2s ease-in-out infinite;
  }
  .focus-ring:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
    border-radius: 8px;
  }
  ::-webkit-scrollbar {
    width: 8px;
  }
  ::-webkit-scrollbar-track {
    background: var(--secondary);
    border-radius: 4px;
  }
  ::-webkit-scrollbar-thumb {
    background: var(--accent);
    border-radius: 4px;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: var(--accent-hover);
  }
  @media (prefers-reduced-motion: reduce) {
    * {
      transition-duration: 0.01ms !important;
      animation-duration: 0.01ms !important;
      animation-iteration-count: 1 !important;
    }
  }
</style>