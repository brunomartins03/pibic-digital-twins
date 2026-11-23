// Reconstructed from the compiled campusFGA scene (the published entity does
// not include .ts sources). Behaviour matches the original Creator Hub script.
import type { Entity } from '@dcl/sdk/ecs';
import { Animator, AudioSource, InputAction, pointerEventsSystem } from '@dcl/sdk/ecs';

export class Button {
  private subs: Record<string, (() => void)[]> = {};

  /**
   * A button that plays its animation and sound when pressed.
   *
   * @param hoverText - Text shown when the player points at the button.
   * @param onClick - Action triggered when the button is pressed.
   */
  constructor(
    public src: string, // DO NOT REMOVE
    public entity: Entity, // DO NOT REMOVE
    public hoverText: string = 'Press',
    public onClick?: () => void,
    public inputButton: InputAction = 1, // IA_PRIMARY
    public sound: string = 'sound.mp3',
    public animation: string = 'trigger',
  ) {}

  /**
   * Start function - called when the script is initialized
   */
  start() {
    if (this.animation) {
      Animator.createOrReplace(this.entity, {
        states: [{ clip: this.animation, playing: false, loop: false }],
      });
    }
    if (this.sound) {
      AudioSource.createOrReplace(this.entity, {
        audioClipUrl: `${this.src}/${this.sound}`,
        playing: false,
      });
    }
    pointerEventsSystem.onPointerDown(
      {
        entity: this.entity,
        opts: { button: this.inputButton, hoverText: this.hoverText, maxDistance: 10 },
      },
      () => this.press(),
    );
  }

  /**
   * Presses the button, playing its animation and sound and triggering its On Click action
   * @action
   */
  public press() {
    if (this.animation) Animator.playSingleAnimation(this.entity, this.animation);
    if (this.sound) AudioSource.playSound(this.entity, `${this.src}/${this.sound}`);
    if (this.onClick) this.onClick();
    for (const callback of this.subs.click ?? []) callback();
  }

  onEvent(event: string, callback: () => void) {
    (this.subs[event] ??= []).push(callback);
  }
}
