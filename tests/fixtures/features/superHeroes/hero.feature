Feature: hero feature
  As a hero
  I want to be able fight so hard
  So that I can save my city from danger

  Background:
    Given laboratory has summoned a local super hero

  Scenario: color of the cloak
    When the superhero flies on the sky
    Then color of its cloak should be "golden"

  Scenario: saviour of the city
    When the superhero is online
    Then citizens should be safe

  Scenario: saviour of the city
    When the superhero is online
    Then citizens should be safe

  Scenario Outline: superheros and their cloaks
    Given laboratory has summoned "<super-hero>"
    When the hero files on the sky
    Then the color of its cloak should be "<color>"
    Examples:
      | super-hero | color  |
      | Thor       | dark   |
      | Iron Man   | purple |
      | Hulk       | green  |

  Scenario: party wizard
    When the superhero and party wizard are together
    Then the city should be filled with parties
