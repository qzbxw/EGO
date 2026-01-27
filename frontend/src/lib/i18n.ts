import { addMessages, init } from 'svelte-i18n';
import en from '../locales/en.json';
import ru from '../locales/ru.json';

addMessages('en', en);
addMessages('ru', ru);

const defaultLocale = 'ru';

init({
	fallbackLocale: defaultLocale,
	initialLocale: defaultLocale
});
