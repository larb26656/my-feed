import { withThemeByClassName } from "@storybook/addon-themes";
import type { Preview } from "@storybook/react-vite";
import "../src/styles.css";

const preview: Preview = {
  decorators: [withThemeByClassName({
    themes: { light: "light", dark: "dark" },
    defaultTheme: "light",
  })],
};

export default preview;
