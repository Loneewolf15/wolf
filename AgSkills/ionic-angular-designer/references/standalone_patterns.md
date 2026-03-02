# Ionic Angular Standalone Patterns

## Core Architecture

Ionic 7.5+ and Angular 17+ prioritize **Standalone Components**. This removes usage of `NgModules` (`@NgModule`) in favor of components that manage their own dependencies.

### 1. Bootstrapping (main.ts)
Instead of `AppModule`, bootstrap the application using `bootstrapApplication` and `provideIonicAngular`.

```typescript
import { bootstrapApplication } from '@angular/platform-browser';
import { RouteReuseStrategy, provideRouter, withPreloading, PreloadAllModules } from '@angular/router';
import { IonicRouteStrategy, provideIonicAngular } from '@ionic/angular/standalone';

import { routes } from './app/app.routes';
import { AppComponent } from './app/app.component';

bootstrapApplication(AppComponent, {
  providers: [
    { provide: RouteReuseStrategy, useClass: IonicRouteStrategy },
    provideIonicAngular(),
    provideRouter(routes, withPreloading(PreloadAllModules)),
  ],
});
```

### 2. Component Structure
Each component must be marked `standalone: true` and imports must be granular.

**DO NOT** import `IonicModule`.
**DO** import specific components from `@ionic/angular/standalone`.

```typescript
import { Component } from '@angular/core';
import { 
  IonHeader, 
  IonToolbar, 
  IonTitle, 
  IonContent, 
  IonButton,
  IonIcon
} from '@ionic/angular/standalone';
import { addIcons } from 'ionicons';
import { heart } from 'ionicons/icons';

@Component({
  selector: 'app-home',
  template: `
    <ion-header>
      <ion-toolbar>
        <ion-title>Home</ion-title>
      </ion-toolbar>
    </ion-header>
    <ion-content class="ion-padding">
      <ion-button>
        <ion-icon name="heart" slot="start"></ion-icon>
        Click Me
      </ion-button>
    </ion-content>
  `,
  standalone: true,
  // Import ONLY what you use
  imports: [IonHeader, IonToolbar, IonTitle, IonContent, IonButton, IonIcon],
})
export class HomePage {
  constructor() {
    // Register icons so 'name="heart"' works
    addIcons({ heart });
  }
}
```

### 3. Routing
Routes are defined as a simple array of route objects.

```typescript
import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: 'home',
    loadComponent: () => import('./home/home.page').then((m) => m.HomePage),
  },
  {
    path: '',
    redirectTo: 'home',
    pathMatch: 'full',
  },
];
```

### 4. Forms
If using forms, import `FormsModule` or `ReactiveFormsModule` directly into the component's `imports` array.

```typescript
import { FormsModule } from '@angular/forms';
import { IonInput, IonItem } from '@ionic/angular/standalone';

@Component({
  // ...
  standalone: true,
  imports: [FormsModule, IonInput, IonItem]
})
```

## Performance Tips
1.  **Lazy Loading**: Always use `loadComponent` in routes.
2.  **Granular Imports**: Never import `IonicModule` (it imports everything). Use `@ionic/angular/standalone`.
3.  **OnPush Change Detection**: Use `changeDetection: ChangeDetectionStrategy.OnPush` where possible for rendering performance.
